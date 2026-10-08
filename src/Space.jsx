import { useEffect, useMemo, useRef, useState } from 'react';
import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';
import { CheckCircle, Circle, SpinnerGap, LockSimple } from '@phosphor-icons/react';
import { groupTasks } from './task-groups.mjs';

const statusIcons = {complete:CheckCircle, active:SpinnerGap, blocked:LockSimple, pending:Circle};
export default function Space({ tasks, selected, onSelect, showLinks, resetKey, view, focusKey, filter, tallPlan=false, grouping='status', groupCounts={} }) {
  const container = useRef(null), canvas = useRef(null), cardLayer=useRef(null), headerLayer=useRef(null), engine=useRef(null);
  const layoutRef=useRef(null);
  const lastHomeInput=useRef({resetKey,view});
  const [failed,setFailed]=useState(false);
  const [size,setSize]=useState({w:1000,h:650});
  const aspect=size.w/size.h;
  const groups = groupTasks(tasks, grouping);
  const longestLane=Math.max(1,...groups.map(group=>group.tasks.length));
  const homeDistance=Math.max(25,44/aspect,longestLane*3.4/(2*Math.tan(Math.PI/9))*1.12);
  const sceneHeight=Math.max(size.h,longestLane*132+120);
  const homeRequested=lastHomeInput.current.resetKey!==resetKey||lastHomeInput.current.view!==view;
  const layoutDistance=useMemo(()=>homeRequested?homeDistance:engine.current?engine.current.camera.position.distanceTo(engine.current.controls.target):homeDistance,[size.w,size.h,resetKey,view,homeRequested,homeDistance,longestLane]);
  const columnSpacing=2*Math.tan(Math.PI/9)*layoutDistance*aspect/groups.length*0.92;
  const nodes = groups.flatMap((lane,col)=>lane.tasks.map((task,row)=>({task, color:lane.color, position:new THREE.Vector3((col-(groups.length-1)/2)*columnSpacing, ((lane.tasks.length-1)/2-row)*3.4, Math.sin(col*1.7+row)*0.38), col, row})));
  const nodeRef=useRef(nodes); nodeRef.current=nodes;
  layoutRef.current={groups,nodes,columnSpacing,tallPlan,sceneHeight};
  const selection=useRef(selected);selection.current=selected;
  const linksEnabled=useRef(showLinks);linksEnabled.current=showLinks;
  const filterRef=useRef(filter);filterRef.current=filter;
  useEffect(()=>{
    const el=container.current;
    const scene=new THREE.Scene();
    scene.fog=new THREE.FogExp2('#080c19',0.008);
    const camera=new THREE.PerspectiveCamera(40,1,0.1,Math.max(5000,homeDistance*5));
    const width=el.clientWidth, height=el.clientHeight;
    camera.aspect=width/height;camera.updateProjectionMatrix();
    const distance=homeDistance;
    camera.position.set(0,3,distance);
    let renderer;
    try {renderer=new THREE.WebGLRenderer({canvas:canvas.current,alpha:true,antialias:true});}
    catch {setFailed(true);return;}
    renderer.setPixelRatio(Math.min(window.devicePixelRatio,1.7,8192/Math.max(width,height)));
    renderer.setSize(width,height);
    renderer.setClearColor('#080c19',0);
    const controls=new OrbitControls(camera,canvas.current);
    controls.enableDamping=true; controls.dampingFactor=0.07;
    controls.minDistance=12; controls.maxDistance=Math.max(300,homeDistance*1.5);
    controls.maxPolarAngle=Math.PI*0.72; controls.minPolarAngle=Math.PI*0.15;
    controls.enablePan=true;
    const grid=new THREE.GridHelper(100,100,'#2a335a','#141b30');
    grid.position.set(0,-8,-6); grid.material.transparent=true;grid.material.opacity=0.42;
    scene.add(grid);
    // An orbital horizon gives the plan a quiet sense of depth.
    const halo=new THREE.Mesh(new THREE.TorusGeometry(17,0.013,6,160),new THREE.MeshBasicMaterial({color:'#3a5474',transparent:true,opacity:0.5}));
    halo.rotation.x=Math.PI/2; halo.position.set(0,-7.95,-3);scene.add(halo);
    const starPositions=new Float32Array(660);
    let seed=19;const rand=()=>{seed=(seed*16807)%2147483647;return(seed-1)/2147483646;};
    for(let i=0;i<starPositions.length;i+=3){starPositions[i]=(rand()-0.5)*95;starPositions[i+1]=(rand()-0.5)*55;starPositions[i+2]=-12-rand()*45;}
    const starsGeometry=new THREE.BufferGeometry();starsGeometry.setAttribute('position',new THREE.BufferAttribute(starPositions,3));
    const stars=new THREE.Points(starsGeometry,new THREE.PointsMaterial({color:'#8b9ac9',size:0.035,transparent:true,opacity:0.6}));scene.add(stars);
    const graph=new THREE.Group();scene.add(graph);
    engine.current={scene,camera,controls,graph,renderer,home:()=>{const w=el.clientWidth,h=el.clientHeight;const rows=Math.max(1,...(layoutRef.current?.groups||[]).map(group=>group.tasks.length));const fit=Math.max(25,44/(w/h),rows*3.4/(2*Math.tan(Math.PI/9))*1.12);camera.far=Math.max(5000,fit*5);camera.updateProjectionMatrix();camera.position.set(0,3,fit);controls.maxDistance=Math.max(300,fit*1.5);controls.target.set(0,0,0);controls.update();}};
    const resize=new ResizeObserver(()=>{const w=el.clientWidth,h=el.clientHeight; camera.aspect=w/h;camera.updateProjectionMatrix();renderer.setPixelRatio(Math.min(window.devicePixelRatio,1.7,8192/Math.max(w,h)));renderer.setSize(w,h);setSize({w,h});});resize.observe(el);
    let frame;
    const render=()=>{
      frame=requestAnimationFrame(render);controls.update();camera.updateMatrixWorld();camera.matrixWorldInverse.copy(camera.matrixWorld).invert();
      const w=el.clientWidth,h=el.clientHeight;
      const layout=layoutRef.current;
      const scaleFactor=layout?.tallPlan?layout.sceneHeight/650:1;
      const minimumScale=0.4;
      const nodes=[...nodeRef.current];
      const cards=[...(cardLayer.current?.querySelectorAll('[data-card]')||[])];
      const labels=[...(headerLayer.current?.querySelectorAll('[data-lane-label]')||[])];
      // Read every DOM measurement before changing any projected style.
      const scrollContainer=container.current?.parentElement;
      const scrollTop=layout?.tallPlan?(scrollContainer?.scrollTop||0):0;
      const labelMetrics=labels.map(label=>{
        const heading=label.querySelector('strong');
        const headingFontSize=heading?parseFloat(window.getComputedStyle(heading).fontSize):12;
        return {headingFontSize:headingFontSize||12};
      });
      const cardPositions=cards.map((card,i)=>{
        const node=nodes[i];if(!node)return null;
        const projected=node.position.clone().project(camera);
        const behind=node.position.clone().applyMatrix4(camera.matrixWorldInverse).z>=0;
        const scale=THREE.MathUtils.clamp(scaleFactor*25/camera.position.distanceTo(node.position),minimumScale,1.4);
        return {projected,behind,scale,zIndex:String(Math.round(1000-projected.z*500))};
      });
      const labelPositions=labels.map((label,i)=>{
        if(!layout)return null;
        const nodeIndex=layout.nodes.findIndex(node=>node.col===i&&node.row===0);
        const node=nodeIndex>=0?layout.nodes[nodeIndex]:null;
        const laneCount=layout.groups.length||1;
        const anchor=node?.position||new THREE.Vector3((i-(laneCount-1)/2)*layout.columnSpacing,3.4,0);
        const projected=anchor.clone().project(camera);
        const behind=anchor.clone().applyMatrix4(camera.matrixWorldInverse).z>=0;
        const cardScale=THREE.MathUtils.clamp(scaleFactor*25/camera.position.distanceTo(anchor),minimumScale,1.4);
        const {headingFontSize}=labelMetrics[i];
        const labelScale=Math.max(cardScale,9/headingFontSize);
        return {
          projected,behind,labelScale,
          left:String((projected.x*0.5+0.5)*w),
          top:String(scrollTop+8),
          zIndex:String(Math.round(1000-projected.z*500))
        };
      });
      // Apply card writes only after all card and label geometry has been read.
      cards.forEach((card,i)=>{
        const placement=cardPositions[i];if(!placement)return;
        card.style.transform='translate(-50%, -50%) scale('+placement.scale+')';
        card.style.left=((placement.projected.x*0.5+0.5)*w)+'px';
        card.style.top=((-placement.projected.y*0.5+0.5)*h)+'px';
        card.style.visibility=placement.behind||placement.projected.z>1?'hidden':'visible';
        card.style.zIndex=placement.zIndex;
      });
      labels.forEach((label,i)=>{
        const placement=labelPositions[i];if(!placement)return;
        label.style.transformOrigin='top center';
        label.style.transform='translateX(-50%) scale('+placement.labelScale+')';
        label.style.left=placement.left+'px';
        label.style.top=placement.top+'px';
        label.style.visibility=placement.behind||placement.projected.z>1?'hidden':'visible';
        label.style.zIndex=placement.zIndex;
      });
      graph.children.forEach(child=>{
        const connected=!selection.current||child.userData.ids?.includes(selection.current);
        if(child.userData.link){child.visible=linksEnabled.current;child.material.opacity=connected?0.8:0.16;}
      });
      renderer.render(scene,camera);
    };render();
    return()=>{cancelAnimationFrame(frame);resize.disconnect();controls.dispose();scene.traverse(o=>{o.geometry?.dispose();if(Array.isArray(o.material))o.material.forEach(m=>m.dispose());else o.material?.dispose();});renderer.dispose();engine.current=null;};
  },[]);
  useEffect(()=>{lastHomeInput.current={resetKey,view};},[resetKey,view]);
  useEffect(()=>{
    const en=engine.current;if(!en)return;
    for(const child of [...en.graph.children]){en.graph.remove(child);child.geometry?.dispose();child.material?.dispose();}
    const byId=new Map(nodes.map(n=>[n.task.id,n]));
    for(const node of nodes){
      const sphere=new THREE.Mesh(new THREE.SphereGeometry(0.075,12,8),new THREE.MeshBasicMaterial({color:node.color}));sphere.position.copy(node.position);sphere.position.y-=0.83;en.graph.add(sphere);
      for(const dependency of node.task.dependencies){
        const from=byId.get(dependency);if(!from)continue;
        const a=from.position.clone().add(new THREE.Vector3(1.8,0,0));
        const b=node.position.clone().add(new THREE.Vector3(-1.8,0,0));
        const curve=new THREE.CubicBezierCurve3(a,a.clone().add(new THREE.Vector3(1.1,0,-1)),b.clone().add(new THREE.Vector3(-1.1,0,-1)),b);
        const line=new THREE.Line(new THREE.BufferGeometry().setFromPoints(curve.getPoints(45)),new THREE.LineBasicMaterial({color:node.color,transparent:true,opacity:0.4}));
        line.userData={link:true,ids:[node.task.id,dependency]};en.graph.add(line);
        const direction=b.clone().sub(curve.getPoint(0.95)).normalize();
        const arrow=new THREE.Mesh(new THREE.ConeGeometry(0.07,0.2,6),new THREE.MeshBasicMaterial({color:node.color,transparent:true,opacity:0.6}));
        arrow.position.copy(b);arrow.quaternion.setFromUnitVectors(new THREE.Vector3(0,1,0),direction);arrow.userData={link:true,ids:[node.task.id,dependency]};en.graph.add(arrow);
      }
    }
  },[tasks,size,grouping,resetKey]);
  useEffect(()=>{engine.current?.home();},[resetKey]);
  useEffect(()=>{
    const en=engine.current;if(!en)return;
    if(view==='map'){const fit=Math.max(25,44/(container.current.clientWidth/container.current.clientHeight),longestLane*3.4/(2*Math.tan(Math.PI/9))*1.12);en.camera.far=Math.max(5000,fit*5);en.camera.updateProjectionMatrix();en.camera.position.set(0,0,fit);en.controls.maxDistance=Math.max(300,fit*1.5);en.controls.target.set(0,0,0);}
    else en.home();
    en.controls.enableRotate=view==='space';en.controls.update();
  },[view,homeDistance,longestLane]);
  useEffect(()=>{if(!focusKey)return;const en=engine.current,node=nodes.find(n=>n.task.id===selected);if(en&&node){const offset=en.camera.position.clone().sub(en.controls.target);en.controls.target.copy(node.position);en.camera.position.copy(node.position.clone().add(offset.multiplyScalar(0.72)));en.controls.update();}},[focusKey]);
  useEffect(()=>{
    if(!tallPlan||!engine.current)return;
    const scene=container.current,area=scene?.parentElement;
    const chosen=focusKey&&selected?nodeRef.current.find(node=>node.task.id===selected):nodeRef.current[0];
    if(!scene||!area||!chosen)return;
    let firstFrame,secondFrame;
    firstFrame=requestAnimationFrame(()=>{secondFrame=requestAnimationFrame(()=>{
      const en=engine.current;if(!en)return;
      en.controls.update();en.camera.updateMatrixWorld();en.camera.matrixWorldInverse.copy(en.camera.matrixWorld).invert();
      const projected=chosen.position.clone().project(en.camera);
      const screenY=(-projected.y*0.5+0.5)*scene.clientHeight;
      const desired=focusKey&&selected?screenY-area.clientHeight/2:screenY-180;
      area.scrollTop=Math.max(0,Math.min(desired,area.scrollHeight-area.clientHeight));
    });});
    return()=>{cancelAnimationFrame(firstFrame);cancelAnimationFrame(secondFrame);};
  },[tallPlan,tasks,resetKey,focusKey,selected,size.w,size.h]);
  const connectedIds=new Set(selected?[selected,...(tasks.find(t=>t.id===selected)?.dependencies||[]),...tasks.filter(t=>t.dependencies.includes(selected)).map(t=>t.id)]:[]);
  return <div className={"space"+(failed?" no-webgl":"")+(tallPlan?" tall-plan":"")} style={tallPlan?{height:sceneHeight+"px"}:undefined} ref={container} onWheelCapture={event=>{if(!tallPlan||event.ctrlKey)return;const area=event.currentTarget.closest(".space-scroll");if(area){area.scrollTop+=event.deltaY;event.stopPropagation();}}}>
    <canvas ref={canvas} aria-label="Interactive 3D task dependency map" />
    <div className="lane-labels" ref={headerLayer}>{groups.map((g,i)=>{const count=groupCounts[g.id]||{showing:g.tasks.length,total:g.tasks.length};return <div data-lane-label key={g.id} style={{'--lane-color':g.color}}><span className="lane-number">0{i+1}</span><strong>{g.title}</strong><span className="lane-count">{count.showing} / {count.total}</span><small>{g.subtitle}</small></div>;})}</div>
    <div className="card-layer" ref={cardLayer}>{nodes.map(({task,color})=>{const Icon=statusIcons[task.status]||Circle;const statusText=task.authoredStatusLabel||(task.status==='complete'&&task.authoredStatus==='checked'?'Marked done':task.status);return <button data-card key={task.id} aria-label={`${task.planLabel?task.planLabel+', ':''}${task.authoredId||task.id}: ${task.title}, ${statusText}`} aria-pressed={selected===task.id} onClick={()=>onSelect(task.id)} className={`task-card ${task.status} ${selected===task.id?'selected':''} ${selected&&!connectedIds.has(task.id)?'muted':''}`} style={{'--lane-color':color}}>
      <div className="card-meta"><span>{task.authoredId||task.id}</span>{task.planLabel&&<small className="task-plan-name" title={task.planLabel}>{task.planLabel}</small>}<Icon size={14} weight={task.status==='complete'?'fill':'regular'}/></div>
      <strong>{task.title}</strong><div className="card-foot"><span>{task.owner||'Unassigned'}</span><span>{task.dependencies.length?`${task.dependencies.length} dependencies`:'Entry point'}</span></div>
    </button>})}</div>
    {failed&&<div className="webgl-error">3D rendering is unavailable in this browser. Use the task list to explore this plan.</div>}
    {!tasks.length&&<div className="space-empty">No tasks in this view. Choose another epic or import a plan.</div>}
  </div>;
}
