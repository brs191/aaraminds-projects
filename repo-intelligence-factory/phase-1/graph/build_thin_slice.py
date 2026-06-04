#!/usr/bin/env python3
"""Emit the hand-verified Phase-1 thin-slice graph as graph.thin-slice.json.

This is the GOLDEN FIXTURE for the resolving extractor: the v1 credit-check call
chain (controller -> routing -> execution -> CSI client -> SOAP), its Spring DI
fan-in, the CSIClient implementors, and the two advising aspects. Every node/edge
line was verified against the repo at SHA 44b6b86. This fixture is the golden oracle
for node/edge TOPOLOGY + provenance (and the sorted byte-stable emit). NOTE: method
ids here use the short `owner#name` form (no overloads exist in this slice); the
production extractor's IdGen appends resolved param FQNs -> `owner#name(paramFQNs)`,
so validate extractor output against this fixture by matching on the `owner#name`
prefix, not the full method id.

It demonstrates: deterministic IDs (no line in the key), full provenance on every
element, confidence tiers (CALLS/IMPLEMENTS exact; INJECTS/ADVISES inferred),
and sorted byte-stable emit. Run: python3 build_thin_slice.py
"""
import json, os

REPO = "apm0045942-credit-routing-service"
SHA  = "44b6b86"
IV   = "extractor-1.1.0"
SRC  = "src/main/java/com/att/creditcheck/"

def sref(subpath, line):
    return f"{REPO}@{SHA}:{SRC}{subpath}:{line}"

nodes, edges = {}, {}

def add_node(nid, label, name, **props):
    n = {"id": nid, "label": label, "name": name,
         "provenance": props.get("provenance", "deterministic"),
         "confidence": props.get("confidence", "exact"),
         "evidence":   props.get("evidence", "ast"),
         "source_ref": props.get("source_ref"),
         "index_version": IV}
    for k in ("kind", "stereotype", "http_method", "path"):
        if k in props: n[k] = props[k]
    nodes[nid] = n
    return nid

def add_edge(etype, src, dst, **props):
    eid = f"edge:{etype}:{src}->{dst}"
    e = {"id": eid, "type": etype, "src": src, "dst": dst,
         "provenance": props.get("provenance", "deterministic"),
         "confidence": props.get("confidence", "exact"),
         "evidence":   props.get("evidence", "ast"),
         "source_ref": props.get("source_ref"),
         "index_version": IV}
    for k in ("call_site", "weave_kind"):
        if k in props: e[k] = props[k]
    edges[eid] = e
    return eid

def tid(fqn):  return "type:" + fqn
def mid(owner, name): return f"method:{owner}#{name}"

# --- version marker (system node, exempt from source_ref) --------------------
add_node(f"buildmeta:{IV}", "BuildMeta", IV, provenance="system",
         confidence="exact", evidence="config", source_ref=None)

# --- Types / Interface / Aspects  (fqn, subpath, line, kind, stereotype) ------
TYPES = [
 ("com.att.creditcheck.routing.v1.CreditRoutingController","routing/v1/CreditRoutingController.java",54,"Class","RestController"),
 ("com.att.creditcheck.routing.v1.CCRoutingService","routing/v1/CCRoutingService.java",46,"Class","Service"),
 ("com.att.creditcheck.routing.v1.CCExecutionService","routing/v1/CCExecutionService.java",47,"Class","Service"),
 ("com.att.creditcheck.routing.v1.CSIRemoteService","routing/v1/CSIRemoteService.java",26,"Class","Service"),
 ("com.att.creditcheck.csi.SoapRequestTransformerService","csi/SoapRequestTransformerService.java",38,"Class","Service"),
 ("com.att.creditcheck.csi.SoapCallService","csi/SoapCallService.java",26,"Class","Service"),
 ("com.att.creditcheck.csi.CSIClient","csi/CSIClient.java",10,"Interface",None),
 ("com.att.creditcheck.routing.v1.aspect.CCRoutingServiceAspect","routing/v1/aspect/CCRoutingServiceAspect.java",45,"Aspect","Aspect"),
 ("com.att.creditcheck.csi.aspect.SoapCallServiceAspect","csi/aspect/SoapCallServiceAspect.java",36,"Aspect","Aspect"),
 # 9 CSIClient implementations
 ("com.att.creditcheck.csi.ecc.ECCServiceClientImpl","csi/ecc/ECCServiceClientImpl.java",24,"Class",None),
 ("com.att.creditcheck.csi.eucc.EUCCServiceClientImpl","csi/eucc/EUCCServiceClientImpl.java",20,"Class",None),
 ("com.att.creditcheck.csi.usocc.USOCCServiceClientImpl","csi/usocc/USOCCServiceClientImpl.java",24,"Class",None),
 ("com.att.creditcheck.csi.iccr.ICCRServiceClientImpl","csi/iccr/ICCRServiceClientImpl.java",24,"Class",None),
 ("com.att.creditcheck.csi.esocc.ESOCCServiceClientImpl","csi/esocc/ESOCCServiceClientImpl.java",24,"Class",None),
 ("com.att.creditcheck.csi.iuccr.IUCCRServiceClientImpl","csi/iuccr/IUCCRServiceClientImpl.java",24,"Class",None),
 ("com.att.creditcheck.csi.cucadp.CUCADPServiceClientImpl","csi/cucadp/CUCADPServiceClientImpl.java",24,"Class",None),
 ("com.att.creditcheck.csi.iucad.IUCADServiceClientImpl","csi/iucad/IUCADServiceClientImpl.java",21,"Class",None),
 ("com.att.creditcheck.csi.account.AddAccountServiceClientImpl","csi/account/AddAccountServiceClientImpl.java",24,"Class",None),
 # CCRoutingService's injected internal dependencies
 ("com.att.creditcheck.admin.rules.CCRuleAdminService","admin/rules/CCRuleAdminService.java",29,"Class","Service"),
 ("com.att.creditcheck.admin.creditcheckresult.CreditCheckResultService","admin/creditcheckresult/CreditCheckResultService.java",46,"Class","Service"),
 ("com.att.creditcheck.admin.ccresultmonitoring.CCResultMonitoringService","admin/ccresultmonitoring/CCResultMonitoringService.java",34,"Class","Service"),
 ("com.att.creditcheck.routing.model.CreditCheckRequestScope","routing/model/CreditCheckRequestScope.java",14,"Class","Component"),
 ("com.att.creditcheck.admin.preapproval.PreApprovalService","admin/preapproval/PreApprovalService.java",42,"Class","Service"),
 ("com.att.creditcheck.admin.eiplimit.EIPLimitService","admin/eiplimit/EIPLimitService.java",42,"Class","Service"),
]
for fqn, sub, line, kind, st in TYPES:
    label = "Aspect" if kind == "Aspect" else ("Type")
    add_node(tid(fqn), label, fqn.rsplit(".",1)[1], kind=kind, stereotype=st,
             source_ref=sref(sub, line))

# external library bean (no in-repo declaration -> provenance external, no source_ref)
add_node(tid("com.fasterxml.jackson.databind.ObjectMapper"), "Type", "ObjectMapper",
         kind="Class", stereotype=None, provenance="external", source_ref=None)

# --- Methods  (owner_fqn, name, line) ----------------------------------------
METHODS = [
 ("com.att.creditcheck.routing.v1.CreditRoutingController","postCreditCheck",92),
 ("com.att.creditcheck.routing.v1.CCRoutingService","routeToCCApi",138),
 ("com.att.creditcheck.routing.v1.CCRoutingService","executeCreditCheck",303),
 ("com.att.creditcheck.routing.v1.CCExecutionService","executeCreditCheck",85),
 ("com.att.creditcheck.routing.v1.CCExecutionService","getCreditCheckResponseHandler",271),
 ("com.att.creditcheck.routing.v1.CSIRemoteService","createSoapRequest",44),
 ("com.att.creditcheck.csi.SoapRequestTransformerService","createSoapRequest",66),
 ("com.att.creditcheck.csi.SoapCallService","getSoapResponse",45),
]
SUB = {fqn: sub for fqn, sub, *_ in TYPES}
for owner, name, line in METHODS:
    add_node(mid(owner, name), "Method", name, kind="Method",
             source_ref=sref(SUB[owner], line))
    add_edge("DEFINES", tid(owner), mid(owner, name), source_ref=sref(SUB[owner], line))

# --- Endpoint ----------------------------------------------------------------
EP = "endpoint:POST /v1/public/api/credit-check"
add_node(EP, "Endpoint", "POST /v1/public/api/credit-check", kind="Endpoint",
         http_method="POST", path="/v1/public/api/credit-check",
         source_ref=sref("routing/v1/CreditRoutingController.java", 52))
add_edge("EXPOSES", tid("com.att.creditcheck.routing.v1.CreditRoutingController"), EP,
         evidence="annotation", source_ref=sref("routing/v1/CreditRoutingController.java", 52))

# --- CALLS (resolved bindings; source_ref = caller decl, call_site = trace) ---
C = "com.att.creditcheck."
CALLS = [
 (mid(C+"routing.v1.CreditRoutingController","postCreditCheck"), mid(C+"routing.v1.CCRoutingService","routeToCCApi"), "routing/v1/CreditRoutingController.java",92,105),
 (mid(C+"routing.v1.CCRoutingService","routeToCCApi"), mid(C+"routing.v1.CCRoutingService","executeCreditCheck"), "routing/v1/CCRoutingService.java",138,164),
 (mid(C+"routing.v1.CCRoutingService","executeCreditCheck"), mid(C+"routing.v1.CCExecutionService","executeCreditCheck"), "routing/v1/CCRoutingService.java",303,315),
 (mid(C+"routing.v1.CCExecutionService","executeCreditCheck"), mid(C+"routing.v1.CCExecutionService","getCreditCheckResponseHandler"), "routing/v1/CCExecutionService.java",85,90),
 (mid(C+"routing.v1.CCExecutionService","getCreditCheckResponseHandler"), mid(C+"routing.v1.CSIRemoteService","createSoapRequest"), "routing/v1/CCExecutionService.java",271,275),
 (mid(C+"routing.v1.CSIRemoteService","createSoapRequest"), mid(C+"csi.SoapRequestTransformerService","createSoapRequest"), "routing/v1/CSIRemoteService.java",44,48),
 (mid(C+"csi.SoapRequestTransformerService","createSoapRequest"), mid(C+"csi.SoapCallService","getSoapResponse"), "csi/SoapRequestTransformerService.java",66,96),
]
for s, d, sub, decl, site in CALLS:
    add_edge("CALLS", s, d, evidence="scip", source_ref=sref(sub, decl),
             call_site=sref(sub, site))

# --- INJECTS (Spring DI; inferred tier) --------------------------------------
def inj(owner_fqn, owner_sub, owner_line, dep_fqn):
    add_edge("INJECTS", tid(owner_fqn), tid(dep_fqn), provenance="inferred",
             confidence="inferred", evidence="annotation", source_ref=sref(owner_sub, owner_line))

inj(C+"routing.v1.CreditRoutingController","routing/v1/CreditRoutingController.java",54, C+"routing.v1.CCRoutingService")
for dep in [C+"admin.rules.CCRuleAdminService", C+"routing.v1.CCExecutionService",
            C+"admin.creditcheckresult.CreditCheckResultService", C+"admin.ccresultmonitoring.CCResultMonitoringService",
            "com.fasterxml.jackson.databind.ObjectMapper", C+"routing.model.CreditCheckRequestScope",
            C+"admin.preapproval.PreApprovalService", C+"admin.eiplimit.EIPLimitService"]:
    inj(C+"routing.v1.CCRoutingService","routing/v1/CCRoutingService.java",82, dep)
for dep in [C+"routing.v1.CSIRemoteService", C+"routing.model.CreditCheckRequestScope"]:
    inj(C+"routing.v1.CCExecutionService","routing/v1/CCExecutionService.java",47, dep)
inj(C+"csi.SoapRequestTransformerService","csi/SoapRequestTransformerService.java",47, C+"csi.SoapCallService")

# --- IMPLEMENTS + each client INJECTS SoapCallService ------------------------
IMPLS = [("ecc.ECCServiceClientImpl",24),("eucc.EUCCServiceClientImpl",20),("usocc.USOCCServiceClientImpl",24),
         ("iccr.ICCRServiceClientImpl",24),("esocc.ESOCCServiceClientImpl",24),("iuccr.IUCCRServiceClientImpl",24),
         ("cucadp.CUCADPServiceClientImpl",24),("iucad.IUCADServiceClientImpl",21),("account.AddAccountServiceClientImpl",24)]
for rel, line in IMPLS:
    fqn = C+"csi."+rel
    sub = "csi/"+rel.replace(".","/")+".java"
    add_edge("IMPLEMENTS", tid(fqn), tid(C+"csi.CSIClient"), source_ref=sref(sub, line))
    inj(fqn, sub, line, C+"csi.SoapCallService")

# --- ADVISES (AOP; inferred tier) --------------------------------------------
add_edge("ADVISES", tid(C+"routing.v1.aspect.CCRoutingServiceAspect"),
         mid(C+"routing.v1.CCRoutingService","routeToCCApi"),
         provenance="inferred", confidence="inferred", evidence="annotation", weave_kind="around",
         source_ref=sref("routing/v1/aspect/CCRoutingServiceAspect.java",72))
add_edge("ADVISES", tid(C+"csi.aspect.SoapCallServiceAspect"),
         mid(C+"csi.SoapCallService","getSoapResponse"),
         provenance="inferred", confidence="inferred", evidence="annotation", weave_kind="around",
         source_ref=sref("csi/aspect/SoapCallServiceAspect.java",54))

# --- emit (sorted by id => byte-stable) --------------------------------------
out = {"index_version": IV, "repo": REPO, "commit": SHA,
       "nodes": [nodes[k] for k in sorted(nodes)],
       "edges": [edges[k] for k in sorted(edges)]}
path = os.path.join(os.path.dirname(__file__), "graph.thin-slice.json")
with open(path, "w") as f:
    json.dump(out, f, indent=2, sort_keys=True)
    f.write("\n")
print(f"wrote {path}: {len(out['nodes'])} nodes, {len(out['edges'])} edges")
