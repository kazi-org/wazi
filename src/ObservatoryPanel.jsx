import {useEffect, useRef, useState} from 'react';
import {hostRequest} from './observatory-api.mjs';

const sectionNames = ['facts','decisions','constraints','intents','openQuestions'];
const sectionLabel = name => name === 'openQuestions' ? 'Open questions' : name[0].toUpperCase()+name.slice(1);

export function ObservatoryPanel({project, plan, task, onNavigate}) {
  const [snapshot,setSnapshot] = useState(null), [context,setContext] = useState(null);
  const [error,setError] = useState(''), [busy,setBusy] = useState(false);
  const [question,setQuestion] = useState('What should I check next for this task?');
  const [answer,setAnswer] = useState(null), [deepError,setDeepError] = useState('');
  const [target,setTarget] = useState('');
  const generation = useRef(0);
  const mounted = useRef(true);
  useEffect(()=>{mounted.current=true;return ()=>{mounted.current=false;};},[]);
  useEffect(()=>{
    const lease=++generation.current, controller=new AbortController();
    setSnapshot(null);setContext(null);setAnswer(null);setError('');setDeepError('');setBusy(false);
    if(!project.hosted)return ()=>controller.abort();
    hostRequest('/api/project',{projectId:project.id},controller.signal).then(value=>{if(generation.current===lease)setSnapshot(value);}).catch(e=>{if(e.name!=='AbortError'&&generation.current===lease)setError(e.message);});
    hostRequest('/api/context',{projectId:project.id},controller.signal).then(value=>{if(generation.current===lease)setContext(value);}).catch(e=>{if(e.name!=='AbortError'&&generation.current===lease)setContext({unavailable:e.message});});
    return ()=>{controller.abort();};
  },[project.id]);
  useEffect(()=>{setAnswer(null);setDeepError('');setBusy(false);},[plan.path,task.id]);
  useEffect(()=>{
    if(!context?.expiresAt)return;
    const delay=Math.max(0,new Date(context.expiresAt).getTime()-Date.now());
    const timer=setTimeout(()=>{setContext({unavailable:'Context lease expired. Refresh before using this context.'});setAnswer(null);},delay);
    return ()=>clearTimeout(timer);
  },[context]);
  if(!project.hosted)return <section className="inspector-section"><h3>PROJECT OBSERVATORY</h3><p className="muted-copy">Select a project discovered by the Go host to analyze local code and inspect scoped context.</p></section>;
  const selection={projectId:project.id,planPath:plan.path,taskId:task.id};
  const proposals=(snapshot?.suggestions||[]).filter(link=>link.planPath===plan.path&&link.taskId===task.id);
  const bindings=(snapshot?.bindings||[]).filter(link=>link.planPath===plan.path&&link.taskId===task.id);
  const bind=async(link,action)=>{
    const lease=generation.current;setError('');
    try{
      await hostRequest('/api/bindings',{...selection,target:link.target,kind:link.kind||'file',basisDigest:link.basisDigest||snapshot.snapshotDigest,sidecarDigest:snapshot.sidecarDigest,action});
      const next=await hostRequest('/api/project',{projectId:project.id});
      if(mounted.current&&generation.current===lease){setSnapshot(next);setTarget('');}
    }catch(e){if(generation.current===lease)setError(e.message);}
  };
  const deeper=async(mode='with-context', regenerate=false)=>{
    const lease=++generation.current;
    setBusy(true);setDeepError('');setAnswer(null);
    try{
      const result=await hostRequest('/api/deep/answer',{...selection,question,contextMode:mode,regenerate,snapshotDigest:snapshot.snapshotDigest,contextLease:context?.leaseId});
      if(mounted.current&&generation.current===lease)setAnswer(result);
    }catch(e){if(generation.current===lease)setDeepError(e.message);}
    finally{if(mounted.current&&generation.current===lease)setBusy(false);}
  };
  return <>
    <section className="inspector-section"><h3>CODE & TEST LINKS</h3>
      {!snapshot&&!error&&<p className="muted-copy">Analyzing this selected project…</p>}
      {error&&<p role="status" className="observatory-error">{error}</p>}
      {snapshot&&<><p className="muted-copy">{snapshot.files?.length||0} bounded local files · presence is not qualified completion.</p>
        {(snapshot.analysis?.warnings||[]).map((warning,i)=><p key={i} className="muted-copy">{warning}</p>)}
        {bindings.map((link,i)=><div className="link-candidate" key={`b${i}`}><strong>{link.target}</strong><small>Confirmed · {link.kind} · {link.freshness||'revalidate against current source'}</small>{(snapshot.bindings||[]).filter(other=>other.target===link.target&&other.taskId!==task.id).map(other=><button key={`${other.planPath}:${other.taskId}`} onClick={()=>onNavigate(other.planPath,other.taskId)}>{other.taskId} · {other.planPath}</button>)}</div>)}
        {proposals.map((link,i)=><div className="link-candidate" key={`s${i}`}><strong>{link.target}</strong><small>Proposed {link.kind} · {link.reason}</small><div className="link-actions"><button onClick={()=>bind(link,'confirm')}>Confirm link</button><button onClick={()=>bind(link,'dismiss')}>Dismiss</button></div></div>)}
        {!proposals.length&&!bindings.length&&<p className="muted-copy">No local match proposed. Add a repository-relative link below.</p>}
        <label className="observatory-label">Manual file or test link<input value={target} onChange={e=>setTarget(e.target.value)} placeholder="src/example.go"/></label><button disabled={!target.trim()} onClick={()=>bind({target:target.trim(),kind:/(test|spec)/i.test(target)?'test':'file'},'manual')}>Confirm manual link</button>
      </>}
    </section>
    <section className="inspector-section"><h3>PROJECT CONTEXT</h3><p className="muted-copy">Read only · explicitly scoped shared brain</p>
      {context?.scope&&<p className="context-scope">Scope: {context.scope.projectId||context.scope.project||'selected project'} · {context.scope.audience||'unqualified audience'}</p>}
      {sectionNames.map(name=>{const section=context?.sections?.[name];return <details className="context-section" key={name}><summary>{sectionLabel(name)} · {section?.status||'unavailable'}</summary>{section?.items?.length?section.items.map((item,i)=><p key={item.id||i}>{item.text||item.content}</p>):<p className="muted-copy">{section?.reason||context?.unavailable||'No qualified owner read API configured for this section.'}</p>}</details>;})}
    </section>
    <section className="inspector-section"><h3>DIG DEEPER</h3><p className="muted-copy">An explicit request sends selected plan/code and the scoped context displayed here to OpenRouter. A provider request may incur a charge.</p>
      <label className="observatory-label">Your question<textarea value={question} onChange={e=>setQuestion(e.target.value)} maxLength={2000}/></label>
      <button disabled={busy||!snapshot||!question.trim()} onClick={()=>deeper()}>Dig deeper{busy?' · requesting…':''}</button>
      <button disabled={busy||!snapshot||!question.trim()} onClick={()=>deeper('plan-code-only')}>Plan/code only · omit brain context</button>
      {deepError&&<p role="status" className="observatory-error">{deepError}</p>}
      {answer&&<div className="deep-answer"><small>{answer.status||'Completed'} · {answer.cached?'Reused identical inputs':'New answer'} · {answer.persistence||'private persistence qualification unavailable'}</small><p>{answer.answer||answer.body||answer.message}</p>{answer.key&&<button onClick={async()=>{try{await hostRequest('/api/deep/delete',{key:answer.key,...selection});setAnswer(null);}catch(e){setDeepError(e.message);}}}>Delete saved answer</button>}<button disabled={busy} onClick={()=>deeper('with-context',true)}>Regenerate · may incur a new charge</button></div>}
    </section>
  </>;
}
