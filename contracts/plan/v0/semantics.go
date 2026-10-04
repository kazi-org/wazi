package v0

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"sort"
	"strings"
)

var validPredicates = setOf("execution-complete", "checks-satisfied", "independent-approved", "landing-verified", "deployment-verified", "domain-accepted")

func semanticValidate(document any, result *Result) {
	definition := get(document, "definition")
	snapshot := get(document, "snapshot")
	tasks := array(get(definition, "tasks"))
	requirements := array(get(definition, "requirements"))
	units := array(get(definition, "executionUnits"))
	evidence := array(get(document, "evidence"))
	evaluations := array(get(document, "evaluations"))

	planID := str(get(definition, "id"))
	planRevision := str(get(definition, "revision"))
	planDigest := str(get(definition, "digest"))
	add := func(code, path, message string) {
		result.Findings = append(result.Findings, Finding{Code: code, Path: path, Message: message})
	}
	if !strings.Contains(planID, ":") {
		add("plan_id_unqualified", "/definition/id", "plan ID must contain an authority namespace separator")
	}
	source := get(definition, "source")
	if str(get(source, "revision")) != planRevision || str(get(source, "digest")) != planDigest {
		add("source_revision_mismatch", "/definition/source", "definition revision and digest must equal the authoritative source revision and digest")
	}
	if !safeSourceRef(str(get(source, "ref"))) {
		add("unsafe_source_ref", "/definition/source/ref", "source ref must be a safe relative path or public HTTPS URI")
	}
	sourceAuthority := str(get(source, "authority"))
	sourceRef := str(get(source, "ref"))

	taskByID := make(map[string]any, len(tasks))
	taskIndex := make(map[string]int, len(tasks))
	for i, task := range tasks {
		id := str(get(task, "id"))
		p := fmt.Sprintf("/definition/tasks/%d", i)
		if !strings.Contains(id, ":") {
			add("task_id_unqualified", p+"/id", "task ID must contain an authority namespace separator")
		}
		if _, exists := taskByID[id]; exists {
			add("duplicate_task_id", p+"/id", "task IDs must be unique")
		}
		taskByID[id], taskIndex[id] = task, i
		taskSource := get(task, "source")
		if str(get(taskSource, "ref")) != sourceRef || !safeSourceRef(str(get(taskSource, "ref"))) {
			add("source_ref_mismatch", p+"/source/ref", "task source ref must safely match the definition source ref")
		}
		if sourceAuthority == "markdown" {
			if get(taskSource, "line") == nil || integer(get(taskSource, "line")) < 1 {
				add("markdown_line_required", p+"/source/line", "Markdown source requires a positive line number")
			}
		} else if sourceAuthority == "native" {
			if str(get(taskSource, "canonicalId")) == "" {
				add("native_canonical_id_required", p+"/source/canonicalId", "native source requires a canonical task ID")
			}
		}
	}

	reqByID := make(map[string]any, len(requirements))
	reqIndex := make(map[string]int, len(requirements))
	for i, req := range requirements {
		id := str(get(req, "id"))
		p := fmt.Sprintf("/definition/requirements/%d", i)
		if _, exists := reqByID[id]; exists {
			add("duplicate_requirement_id", p+"/id", "requirement IDs must be unique")
		}
		reqByID[id], reqIndex[id] = req, i
		if _, ok := taskByID[str(get(req, "taskId"))]; !ok {
			add("dangling_requirement_task", p+"/taskId", "requirement task must exist")
		}
		if str(get(req, "policyRevision")) == "" {
			add("policy_revision_required", p+"/policyRevision", "requirement must pin a policy revision")
		}
		if len(object(get(req, "subject"))) == 0 {
			add("subject_required", p+"/subject", "requirement must name a nonempty subject")
		}
		if str(get(req, "predicate")) == "domain-accepted" && !strings.Contains(str(get(req, "domain")), ":") {
			add("domain_unqualified", p+"/domain", "domain acceptance requires a namespaced domain")
		}
	}
	_ = reqIndex

	// Validate all graph edges before checking for cycles.
	graph := make(map[string][]string, len(tasks))
	for i, task := range tasks {
		tid := str(get(task, "id"))
		for j, rawDep := range array(get(task, "dependencies")) {
			dep := str(get(rawDep, "taskId"))
			p := fmt.Sprintf("/definition/tasks/%d/dependencies/%d", i, j)
			if dep == tid {
				add("self_dependency", p+"/taskId", "task cannot depend on itself")
			}
			if _, ok := taskByID[dep]; !ok {
				add("dangling_dependency", p+"/taskId", "dependency task does not exist")
			} else {
				graph[tid] = append(graph[tid], dep)
			}
			predicate := str(get(rawDep, "predicate"))
			if _, ok := validPredicates[predicate]; !ok {
				add("invalid_dependency_predicate", p+"/predicate", "dependency predicate is not supported")
			}
			reqID := str(get(rawDep, "requirementId"))
			if predicate != "execution-complete" && reqID == "" {
				add("dependency_requirement_required", p+"/requirementId", "non-execution dependency must identify a requirement")
			}
			if reqID != "" {
				req, ok := reqByID[reqID]
				if !ok {
					add("dangling_dependency_requirement", p+"/requirementId", "dependency requirement does not exist")
				} else if str(get(req, "taskId")) != dep || str(get(req, "predicate")) != predicate {
					add("dependency_requirement_mismatch", p+"/requirementId", "dependency requirement must belong to its task and use the same predicate")
				}
			}
		}
	}
	checkCycles(graph, taskIndex, add)

	unitByKey := map[string]any{}
	unitByID := map[string]any{}
	taskUnit := map[string]string{}
	for i, unit := range units {
		p := fmt.Sprintf("/definition/executionUnits/%d", i)
		authority, canonicalID := str(get(unit, "authority")), str(get(unit, "canonicalId"))
		key := authority + "\x00" + canonicalID
		if _, exists := unitByKey[key]; exists {
			add("duplicate_execution_unit", p, "(authority, canonicalId) execution unit keys must be unique")
		}
		unitByKey[key] = unit
		unitID := str(get(unit, "id"))
		if _, exists := unitByID[unitID]; exists {
			add("duplicate_execution_unit_id", p+"/id", "execution unit IDs must be unique")
		}
		unitByID[unitID] = unit
		covered := setOf()
		stages := setOf(stringsOf(get(unit, "stages"))...)
		for j, rawID := range stringsOf(get(unit, "taskIds")) {
			if _, duplicate := covered[rawID]; duplicate {
				add("duplicate_unit_task", fmt.Sprintf("%s/taskIds/%d", p, j), "execution unit task coverage must be unique")
			}
			covered[rawID] = struct{}{}
			task, ok := taskByID[rawID]
			if !ok {
				add("dangling_unit_task", fmt.Sprintf("%s/taskIds/%d", p, j), "execution unit task does not exist")
				continue
			}
			if previous, duplicate := taskUnit[rawID]; duplicate && previous != key {
				add("multiple_execution_units", fmt.Sprintf("%s/taskIds/%d", p, j), "a task can belong to at most one execution unit")
			}
			taskUnit[rawID] = key
			if _, ok := stages[str(get(task, "stage"))]; !ok {
				add("unit_stage_missing", p+"/stages", "execution unit must list every covered task stage")
			}
		}
	}
	for i, task := range tasks {
		tid := str(get(task, "id"))
		reference := get(task, "executionUnit")
		referenceID := str(get(task, "executionUnitId"))
		if referenceID == "" {
			if _, covered := taskUnit[tid]; covered {
				add("unit_task_reference_missing", fmt.Sprintf("/definition/tasks/%d", i), "task and execution unit task lists must agree")
			}
			continue
		}
		unit, ok := unitByKeyByID(units, referenceID)
		key := str(get(unit, "authority")) + "\x00" + str(get(unit, "canonicalId"))
		if !ok {
			add("unknown_task_unit", fmt.Sprintf("/definition/tasks/%d/executionUnit", i), "task execution unit reference does not resolve")
		} else if taskUnit[tid] != key {
			add("unit_coverage_mismatch", fmt.Sprintf("/definition/tasks/%d/executionUnit", i), "task reference and unit task list must agree")
		}
	}

	if snapshot != nil {
		if str(get(snapshot, "planId")) != planID || str(get(snapshot, "planRevision")) != planRevision || str(get(snapshot, "planDigest")) != planDigest {
			add("snapshot_plan_mismatch", "/snapshot", "snapshot plan identity, revision and digest must match the definition")
		}
	}
	execByTask := map[string]any{}
	for i, exec := range array(get(snapshot, "executions")) {
		taskID := str(get(exec, "taskId"))
		p := fmt.Sprintf("/snapshot/executions/%d", i)
		if _, ok := taskByID[taskID]; !ok {
			add("dangling_execution_task", p+"/taskId", "execution task does not exist")
		}
		if _, duplicate := execByTask[taskID]; duplicate {
			add("duplicate_current_execution", p+"/taskId", "snapshot has more than one current execution per task")
		}
		execByTask[taskID] = exec
		expectedUnit, inUnit := taskUnit[taskID]
		actualUnit := str(get(exec, "executionUnitId"))
		if inUnit {
			if actualUnit == "" || actualUnit != str(get(taskByID[taskID], "executionUnitId")) || actualUnit != unitIDByKey(units, expectedUnit) {
				add("execution_unit_mismatch", p+"/executionUnit", "snapshot execution unit must match authored task enrollment")
			}
		} else if actualUnit != "" {
			add("execution_unit_unenrolled", p+"/executionUnit", "snapshot cannot enroll a task into an unauthored unit")
		}
	}

	evidenceByID := make(map[string]any, len(evidence))
	for i, item := range evidence {
		id := str(get(item, "id"))
		if _, duplicate := evidenceByID[id]; duplicate {
			add("duplicate_evidence_id", fmt.Sprintf("/evidence/%d/id", i), "evidence IDs must be unique")
		}
		evidenceByID[id] = item
		if !safeSourceRef(str(get(item, "provenance"))) {
			add("unsafe_evidence_provenance", fmt.Sprintf("/evidence/%d/provenance", i), "evidence provenance must be a logical URN, safe relative path or public HTTPS URI")
		}
		independence := get(item, "independence")
		if independence != nil && str(get(independence, "provenance")) != "" && !safeSourceRef(str(get(independence, "provenance"))) {
			add("unsafe_attestation_provenance", fmt.Sprintf("/evidence/%d/independence/provenance", i), "attestation provenance must be a logical URN, safe relative path or public HTTPS URI")
		}
		if _, ok := taskByID[str(get(item, "taskId"))]; !ok {
			add("dangling_evidence_task", fmt.Sprintf("/evidence/%d/taskId", i), "evidence task does not exist")
		}
		if !truth(get(item, "auditOnly")) {
			if str(get(item, "planRevision")) != planRevision || str(get(item, "planDigest")) != planDigest {
				add("evidence_revision_mismatch", fmt.Sprintf("/evidence/%d", i), "current evidence must match definition revision and digest")
			}
			if snapshot == nil || execByTask[str(get(item, "taskId"))] == nil || str(get(item, "attemptId")) != str(get(execByTask[str(get(item, "taskId"))], "attemptId")) {
				add("evidence_attempt_mismatch", fmt.Sprintf("/evidence/%d/attemptId", i), "current evidence must match the task's current snapshot attempt")
			}
		}
		state := str(get(execByTask[str(get(item, "taskId"))], "state"))
		if str(get(item, "kind")) == "execution" && (state == "canceled" || state == "expired") && !truth(get(item, "auditOnly")) {
			add("terminal_execution_evidence", fmt.Sprintf("/evidence/%d", i), "canceled or expired execution evidence must be audit-only")
		}
	}

	evaluationIDs := setOf()
	logicalKeys := setOf()
	for i, evaluation := range evaluations {
		p := fmt.Sprintf("/evaluations/%d", i)
		id := str(get(evaluation, "id"))
		if _, duplicate := evaluationIDs[id]; duplicate {
			add("duplicate_evaluation_id", p+"/id", "evaluation IDs must be unique")
		}
		evaluationIDs[id] = struct{}{}
		taskID, reqID := str(get(evaluation, "taskId")), str(get(evaluation, "requirementId"))
		req, ok := reqByID[reqID]
		if !ok {
			add("dangling_evaluation_requirement", p+"/requirementId", "evaluation requirement does not exist")
		} else if str(get(req, "taskId")) != taskID {
			add("evaluation_requirement_mismatch", p+"/requirementId", "evaluation task must match requirement task")
		}
		if _, ok := taskByID[taskID]; !ok {
			add("dangling_evaluation_task", p+"/taskId", "evaluation task does not exist")
		}
		logical := strings.Join([]string{reqID, taskID, str(get(evaluation, "attemptId")), str(get(evaluation, "planRevision")), str(get(evaluation, "policyRevision"))}, "\x00")
		if _, duplicate := logicalKeys[logical]; duplicate {
			add("duplicate_evaluation_key", p, "evaluation logical key must be unique for a requirement, task, attempt, plan revision and policy revision")
		}
		logicalKeys[logical] = struct{}{}
		if ok && str(get(evaluation, "policyRevision")) != str(get(req, "policyRevision")) {
			add("evaluation_policy_mismatch", p+"/policyRevision", "evaluation policy revision must match requirement")
		}
		ids := stringsOf(get(evaluation, "evidenceIds"))
		referenced := map[string]any{}
		for j, evidenceID := range ids {
			item, exists := evidenceByID[evidenceID]
			if !exists {
				add("dangling_evaluation_evidence", fmt.Sprintf("%s/evidenceIds/%d", p, j), "evaluation evidence reference does not resolve")
				continue
			}
			referenced[evidenceID] = item
		}
		if str(get(evaluation, "outcome")) != "satisfied" {
			continue
		}
		if str(get(evaluation, "qualification")) != "verified" {
			add("satisfied_not_verified", p+"/qualification", "satisfied evaluation must be qualified verified")
		}
		if !ok {
			continue
		}
		if str(get(get(definition, "metadata"), "sourceState")) == "dirty" {
			add("dirty_source_satisfied", p, "dirty authored source cannot support a satisfied current evaluation")
		}
		if snapshot == nil {
			add("satisfied_without_snapshot", p, "satisfied evaluation requires a current snapshot")
			continue
		}
		exec := execByTask[taskID]
		if exec == nil || str(get(exec, "attemptId")) != str(get(evaluation, "attemptId")) || str(get(exec, "planRevision")) != "" && str(get(exec, "planRevision")) != planRevision {
			add("satisfied_attempt_mismatch", p+"/attemptId", "satisfied evaluation must bind the task's current attempt")
		}
		if str(get(evaluation, "planRevision")) != planRevision || str(get(evaluation, "planDigest")) != planDigest {
			add("satisfied_plan_mismatch", p, "satisfied evaluation must bind exact current plan revision and digest")
		}
		state := str(get(exec, "state"))
		if state == "canceled" || state == "expired" || state == "failed" {
			add("terminal_execution_satisfied", p, "terminal task execution cannot be revived by a satisfied evaluation")
		}
		predicate := str(get(req, "predicate"))
		if predicate == "execution-complete" && state != "complete" {
			add("execution_not_complete", p, "execution completion requires current execution state complete")
		}
		if (predicate == "checks-satisfied" || predicate == "independent-approved" || predicate == "landing-verified") && (str(get(get(req, "subject"), "head")) == "" || str(get(get(req, "subject"), "base")) == "") {
			add("subject_head_base_required", p, "checks, review and landing require head and base")
		}
		if predicate == "deployment-verified" && (str(get(get(req, "subject"), "artifact")) == "" || str(get(get(req, "subject"), "environment")) == "") {
			add("deployment_subject_required", p, "deployment requires artifact and environment")
		}
		if predicate == "domain-accepted" && str(get(get(req, "subject"), "artifact")) == "" {
			add("domain_artifact_required", p, "domain acceptance requires an artifact")
		}
		if predicate == "domain-accepted" && !strings.Contains(str(get(req, "domain")), ":") {
			add("domain_unqualified", p, "domain acceptance requires a namespaced domain")
		}

		qualifying := 0
		for _, evidenceID := range ids {
			item := referenced[evidenceID]
			if item == nil || !qualifyingEvidence(item, req, evaluation, snapshot, exec) {
				continue
			}
			if kindMatches(predicate, str(get(item, "kind"))) {
				qualifying++
			}
		}
		if qualifying == 0 {
			add("qualifying_evidence_missing", p+"/evidenceIds", "satisfied evaluation requires at least one qualifying evidence item of the predicate's kind")
		}
		if predicate == "independent-approved" {
			checkIndependence(ids, referenced, p, add)
		}
		if predicate == "deployment-verified" {
			approvalID := str(get(evaluation, "approvalEvidenceId"))
			approval := referenced[approvalID]
			if approvalID == "" || approval == nil || str(get(approval, "kind")) != "policy-approval" || !qualifyingEvidence(approval, req, evaluation, snapshot, exec) || str(get(get(approval, "metadata"), "scope")) != "deployment" {
				add("deployment_approval_missing", p+"/approvalEvidenceId", "deployment satisfaction requires a referenced verified policy approval scoped to deployment")
			}
		}
		if get(evaluation, "alternative") != nil {
			if predicate != "checks-satisfied" {
				add("unsupported_alternative", p+"/alternative", "alternative policy applies only to checks-satisfied")
			} else {
				checkAlternative(get(evaluation, "alternative"), req, ids, referenced, evaluation, p, add)
			}
		}
		if predicate == "checks-satisfied" && hasUnavailableCheck(ids, referenced) && get(evaluation, "alternative") == nil {
			add("unavailable_check_without_alternative", p, "unavailable hosted check requires an explicitly approved local alternative")
		}
	}
}

func qualifyingEvidence(item, requirement, evaluation, snapshot, execution any) bool {
	if truth(get(item, "auditOnly")) || str(get(item, "trust")) != "verified" || str(get(item, "result")) != "passed" {
		return false
	}
	if str(get(item, "taskId")) != str(get(evaluation, "taskId")) || str(get(item, "attemptId")) != str(get(evaluation, "attemptId")) || str(get(item, "planRevision")) != str(get(evaluation, "planRevision")) || str(get(item, "planDigest")) != str(get(evaluation, "planDigest")) || str(get(item, "policyRevision")) != str(get(requirement, "policyRevision")) {
		return false
	}
	if str(get(evaluation, "policyRevision")) != str(get(item, "policyRevision")) {
		return false
	}
	if !subjectMatches(get(requirement, "subject"), get(snapshot, "subject")) || !subjectMatches(get(requirement, "subject"), get(item, "subject")) {
		return false
	}
	if str(get(item, "kind")) == "execution" && (str(get(execution, "state")) == "canceled" || str(get(execution, "state")) == "expired") {
		return false
	}
	return true
}

func checkAlternative(alternative, requirement any, ids []string, referenced map[string]any, evaluation any, path string, add func(string, string, string)) {
	p := path + "/alternative"
	if !truth(get(requirement, "allowLocalAlternative")) {
		add("alternative_not_allowed", p, "requirement must explicitly allow a local alternative")
	}
	if str(get(alternative, "scope")) != "source-merge" {
		add("alternative_scope_invalid", p+"/scope", "local alternative scope must be source-merge")
	}
	approvalID := str(get(alternative, "approvalEvidenceId"))
	approval := referenced[approvalID]
	if !contains(ids, approvalID) || approval == nil || str(get(approval, "kind")) != "policy-approval" || str(get(approval, "trust")) != "verified" || str(get(approval, "result")) != "passed" || truth(get(approval, "auditOnly")) || str(get(get(approval, "metadata"), "scope")) != "source-merge" || !evidenceMatchesEvaluation(approval, requirement, evaluation) {
		add("alternative_approval_invalid", p+"/approvalEvidenceId", "alternative must reference verified, passed source-merge policy approval bound to the same claim")
	}
	if !hasUnavailableCheck(ids, referenced) {
		add("alternative_unavailable_observation_missing", p, "alternative must retain an unavailable hosted check observation")
	}
	localPassed := false
	for _, id := range ids {
		item := referenced[id]
		if item != nil && str(get(item, "kind")) == "check" && str(get(item, "result")) == "passed" && qualifyingEvidence(item, requirement, evaluation, map[string]any{"subject": get(item, "subject")}, nil) {
			localPassed = true
		}
	}
	if !localPassed {
		add("alternative_local_observation_missing", p, "alternative must include a verified passed local check")
	}
}

func hasUnavailableCheck(ids []string, referenced map[string]any) bool {
	for _, id := range ids {
		item := referenced[id]
		if item != nil && str(get(item, "kind")) == "check" && str(get(item, "result")) == "unavailable" && !truth(get(item, "auditOnly")) {
			return true
		}
	}
	return false
}

func evidenceMatchesEvaluation(item, requirement, evaluation any) bool {
	return str(get(item, "taskId")) == str(get(evaluation, "taskId")) && str(get(item, "attemptId")) == str(get(evaluation, "attemptId")) && str(get(item, "planRevision")) == str(get(evaluation, "planRevision")) && str(get(item, "planDigest")) == str(get(evaluation, "planDigest")) && str(get(item, "policyRevision")) == str(get(requirement, "policyRevision")) && str(get(item, "policyRevision")) == str(get(evaluation, "policyRevision")) && subjectMatches(get(requirement, "subject"), get(item, "subject"))
}

func subjectMatches(required, actual any) bool {
	r := object(required)
	a := object(actual)
	for k, v := range r {
		if av, ok := a[k]; !ok || fmt.Sprint(av) != fmt.Sprint(v) {
			return false
		}
	}
	return len(r) > 0
}

func kindMatches(predicate, kind string) bool {
	want := map[string]string{"execution-complete": "execution", "checks-satisfied": "check", "independent-approved": "review", "landing-verified": "landing", "deployment-verified": "deployment", "domain-accepted": "domain"}
	return want[predicate] == kind
}

func checkIndependence(ids []string, referenced map[string]any, path string, add func(string, string, string)) {
	for _, id := range ids {
		item := referenced[id]
		if item == nil || str(get(item, "kind")) != "review" || str(get(item, "result")) != "passed" || str(get(item, "trust")) != "verified" || truth(get(item, "auditOnly")) {
			continue
		}
		ind := get(item, "independence")
		mode := str(get(ind, "mode"))
		verifier := str(get(item, "verifier"))
		contributors := stringsOf(get(item, "contributors"))
		switch mode {
		case "contributors":
			if len(contributors) == 0 || contains(contributors, verifier) {
				add("independence_contributors_invalid", path+"/evidenceIds", "contributors independence needs nonempty contributors excluding the verifier")
			}
			return
		case "authority-attestation":
			if str(get(ind, "authority")) == "" || str(get(ind, "provenance")) == "" || contains(contributors, verifier) {
				add("independence_attestation_invalid", path+"/evidenceIds", "authority attestation needs named authority and provenance and must exclude the verifier when listed")
			}
			return
		case "singular":
			add("independence_singular", path+"/evidenceIds", "singular mode cannot establish all-contributor independence")
			return
		default:
			add("independence_missing", path+"/evidenceIds", "independent approval requires explicit independence evidence")
			return
		}
	}
	add("independence_review_missing", path+"/evidenceIds", "independent approval requires qualifying review evidence")
}

func checkCycles(graph map[string][]string, index map[string]int, add func(string, string, string)) {
	state := map[string]uint8{}
	var visit func(string)
	visit = func(id string) {
		if state[id] == 2 {
			return
		}
		if state[id] == 1 {
			add("dependency_cycle", fmt.Sprintf("/definition/tasks/%d/dependencies", index[id]), "dependency graph contains a cycle")
			return
		}
		state[id] = 1
		for _, next := range graph[id] {
			visit(next)
		}
		state[id] = 2
	}
	ids := make([]string, 0, len(graph))
	for id := range graph {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		visit(id)
	}
}

func safeSourceRef(value string) bool {
	if value == "" || strings.ContainsAny(value, "\\\x00\n\r") || strings.HasPrefix(value, "//") {
		return false
	}
	u, err := url.Parse(value)
	if err != nil {
		return false
	}
	if u.Scheme == "" {
		if u.Host != "" || strings.HasPrefix(value, "/") || u.RawQuery != "" || u.Fragment != "" {
			return false
		}
		for _, segment := range strings.Split(u.Path, "/") {
			if segment == ".." || segment == "." || segment == "" {
				return false
			}
		}
		return true
	}
	if strings.EqualFold(u.Scheme, "urn") {
		return u.Opaque != "" && u.Host == "" && u.User == nil && !strings.ContainsAny(u.Opaque, " \t\r\n")
	}
	if !strings.EqualFold(u.Scheme, "https") || u.Host == "" || u.User != nil || u.Opaque != "" {
		return false
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return false
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		ip = ip.Unmap()
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
			return false
		}
	} else if net.ParseIP(host) != nil {
		return false
	}
	return true
}

func integer(value any) int {
	if n, ok := value.(interface{ Int64() (int64, error) }); ok {
		v, _ := n.Int64()
		return int(v)
	}
	return 0
}

func setOf(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, v := range values {
		out[v] = struct{}{}
	}
	return out
}

func unitByKeyByID(units []any, id string) (any, bool) {
	for _, unit := range units {
		if str(get(unit, "id")) == id {
			return unit, true
		}
	}
	return nil, false
}

func unitIDByKey(units []any, key string) string {
	for _, unit := range units {
		if str(get(unit, "authority"))+"\x00"+str(get(unit, "canonicalId")) == key {
			return str(get(unit, "id"))
		}
	}
	return ""
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
