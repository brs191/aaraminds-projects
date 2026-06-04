package com.aaraminds.repointel;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.javaparser.ParserConfiguration;
import com.github.javaparser.ast.CompilationUnit;
import com.github.javaparser.ast.body.*;
import com.github.javaparser.ast.expr.MethodCallExpr;
import com.github.javaparser.resolution.declarations.ResolvedMethodDeclaration;
import com.github.javaparser.symbolsolver.JavaSymbolSolver;
import com.github.javaparser.symbolsolver.resolution.typesolvers.*;
import com.github.javaparser.utils.SourceRoot;

import java.io.File;
import java.net.URL;
import java.net.URLClassLoader;
import java.nio.file.*;
import java.util.*;

/**
 * Phase-1 deterministic resolving extractor (build-order step 3).
 *
 * Parses a BUILT Spring Boot checkout with a RESOLVING symbol solver (JavaParser +
 * JavaSymbolSolver over source roots + the compiled classpath), then emits the M0
 * graph (SCHEMA.md): Type/Method nodes, DEFINES/EXTENDS/IMPLEMENTS/CALLS (Tier-A,
 * resolved bindings), Spring stereotypes + INJECTS (Tier-C), each carrying
 * repo@sha:path:line provenance. Output is sorted by id => byte-stable.
 *
 * Not a syntax-only parser: every CALLS edge comes from mce.resolve(), never a name
 * match (constraint §3.1). Lombok/JAXB members resolve because target/classes is on
 * the solver classpath (build integration, §3.5).
 *
 * NOTE: authored against JavaParser 3.26; build + run in a JDK 17 + Maven env
 * (run.sh). It was not compiled in the research sandbox (JDK 11 JRE only).
 */
public final class Extractor {

    static final String IV = "extractor-1.1.0";
    static final Set<String> STEREO = new HashSet<>(Arrays.asList(
            "RestController", "Controller", "Service", "Repository", "Component", "Configuration", "Aspect"));

    final String repo, sha, basePkg;
    final ObjectMapper om = new ObjectMapper();
    final TreeMap<String, Map<String, Object>> nodes = new TreeMap<>();
    final TreeMap<String, Map<String, Object>> edges = new TreeMap<>();
    int unresolved = 0;

    Extractor(String repo, String sha, String basePkg) { this.repo = repo; this.sha = sha; this.basePkg = basePkg; }

    // ---- emit helpers --------------------------------------------------------
    String sref(CompilationUnit cu, int line) {
        String p = cu.getStorage().map(s -> s.getPath().toString()).orElse("?");
        for (String anchor : new String[]{"/src/main/java/", "/target/generated-sources/"}) {
            int i = p.indexOf(anchor);
            if (i >= 0) { p = p.substring(i + 1); break; }
        }
        return repo + "@" + sha + ":" + p + ":" + line;
    }
    void node(String id, String label, String name, String prov, String conf, String ev, String sref, Map<String, Object> extra) {
        if (nodes.containsKey(id)) return;
        Map<String, Object> n = new LinkedHashMap<>();
        n.put("id", id); n.put("label", label); n.put("name", name);
        n.put("provenance", prov); n.put("confidence", conf); n.put("evidence", ev);
        n.put("source_ref", sref); n.put("index_version", IV);
        if (extra != null) n.putAll(extra);
        nodes.put(id, n);
    }
    void externalType(String fqn) {
        node(IdGen.type(fqn), "Type", simple(fqn), "external", "exact", "ast", null,
                Collections.singletonMap("kind", "Class"));
    }
    void edge(String type, String src, String dst, String prov, String conf, String ev, String sref, Map<String, Object> extra) {
        String id = IdGen.edge(type, src, dst);
        if (edges.containsKey(id)) return;
        Map<String, Object> e = new LinkedHashMap<>();
        e.put("id", id); e.put("type", type); e.put("src", src); e.put("dst", dst);
        e.put("provenance", prov); e.put("confidence", conf); e.put("evidence", ev);
        e.put("source_ref", sref); e.put("index_version", IV);
        if (extra != null) e.putAll(extra);
        edges.put(id, e);
    }
    static String simple(String fqn) { int i = fqn.lastIndexOf('.'); return i < 0 ? fqn : fqn.substring(i + 1); }
    boolean inProject(String fqn) { return fqn != null && fqn.startsWith(basePkg); }

    String paramSig(CallableDeclaration<?> m) {
        List<String> ps = new ArrayList<>();
        for (Parameter p : m.getParameters()) {
            try { ps.add(p.getType().resolve().describe()); }
            catch (Throwable t) { ps.add(p.getTypeAsString()); }
        }
        return String.join(",", ps);
    }
    String stereotype(TypeDeclaration<?> t) {
        for (var a : t.getAnnotations()) if (STEREO.contains(a.getNameAsString())) return a.getNameAsString();
        return null;
    }

    // ---- pass 1: declarations + structural edges -----------------------------
    void declarations(CompilationUnit cu) {
        for (ClassOrInterfaceDeclaration t : cu.findAll(ClassOrInterfaceDeclaration.class)) {
            String fqn = t.getFullyQualifiedName().orElse(null);
            if (fqn == null) continue;
            String label = t.isInterface() ? "Type" : (hasAspect(t) ? "Aspect" : "Type");
            String stereo = t.isInterface() ? null : stereotype(t);
            Map<String, Object> ex = new LinkedHashMap<>();
            ex.put("kind", t.isInterface() ? "Interface" : "Class");
            if (stereo != null) ex.put("stereotype", stereo);
            node(IdGen.type(fqn), label, t.getNameAsString(), "deterministic", "exact", "ast",
                    sref(cu, t.getBegin().map(p -> p.line).orElse(0)), ex);

            // EXTENDS / IMPLEMENTS (resolved to declared types)
            for (var sup : t.getExtendedTypes())   structEdge(cu, t, fqn, sup.resolve().describe(), "EXTENDS");
            for (var imp : t.getImplementedTypes()) structEdge(cu, t, fqn, safeResolve(imp), "IMPLEMENTS");

            // methods + constructors -> Method nodes + DEFINES
            List<CallableDeclaration<?>> callables = new ArrayList<>();
            callables.addAll(t.getMethods()); callables.addAll(t.getConstructors());
            for (CallableDeclaration<?> m : callables) {
                String mid = IdGen.method(fqn, m.getNameAsString(), paramSig(m));
                node(mid, "Method", m.getNameAsString(), "deterministic", "exact", "ast",
                        sref(cu, m.getBegin().map(p -> p.line).orElse(0)),
                        Collections.singletonMap("kind", m instanceof ConstructorDeclaration ? "Constructor" : "Method"));
                edge("DEFINES", IdGen.type(fqn), mid, "deterministic", "exact", "ast",
                        sref(cu, m.getBegin().map(p -> p.line).orElse(0)), null);
            }
            // Spring DI: constructor-param + @Autowired field injection -> INJECTS (Tier-C)
            if (stereo != null) injects(cu, t, fqn);
        }
    }
    boolean hasAspect(TypeDeclaration<?> t) { for (var a : t.getAnnotations()) if (a.getNameAsString().equals("Aspect")) return true; return false; }
    String safeResolve(com.github.javaparser.ast.type.ClassOrInterfaceType ty) {
        try { return ty.resolve().describe(); } catch (Throwable e) { return basePkg + "." + ty.getNameAsString(); }
    }
    void structEdge(CompilationUnit cu, TypeDeclaration<?> t, String fromFqn, String toFqn, String type) {
        if (toFqn == null) return;
        if (!nodes.containsKey(IdGen.type(toFqn))) externalType(toFqn);
        edge(type, IdGen.type(fromFqn), IdGen.type(toFqn), "deterministic", "exact", "ast",
                sref(cu, t.getBegin().map(p -> p.line).orElse(0)), null);
    }
    void injects(CompilationUnit cu, ClassOrInterfaceDeclaration t, String fqn) {
        int line = t.getBegin().map(p -> p.line).orElse(0);
        // single/constructor injection
        for (ConstructorDeclaration c : t.getConstructors())
            for (Parameter p : c.getParameters()) injectEdge(cu, fqn, p.getType(), line);
        // @Autowired fields (incl Lombok @RequiredArgsConstructor over final fields)
        for (FieldDeclaration f : t.getFields())
            if (f.isFinal() || f.getAnnotations().stream().anyMatch(a -> a.getNameAsString().equals("Autowired")))
                injectEdge(cu, fqn, f.getElementType(), line);
    }
    void injectEdge(CompilationUnit cu, String ownerFqn, com.github.javaparser.ast.type.Type ty, int line) {
        String dep; try { dep = ty.resolve().describe(); } catch (Throwable e) { return; }
        if (dep.startsWith("java.")) return; // ignore JDK scalars
        if (!nodes.containsKey(IdGen.type(dep))) externalType(dep);
        edge("INJECTS", IdGen.type(ownerFqn), IdGen.type(dep), "inferred", "inferred", "annotation",
                sref(cu, line), null);
    }

    // ---- pass 2: resolved CALLS ---------------------------------------------
    void calls(CompilationUnit cu) {
        for (MethodCallExpr mce : cu.findAll(MethodCallExpr.class)) {
            Optional<CallableDeclaration> encl = mce.findAncestor(CallableDeclaration.class);
            if (encl.isEmpty()) continue;
            TypeDeclaration<?> ot = (TypeDeclaration<?>) encl.get().findAncestor(TypeDeclaration.class).orElse(null);
            if (ot == null || ot.getFullyQualifiedName().isEmpty()) continue;
            String srcId = IdGen.method(ot.getFullyQualifiedName().get(), encl.get().getNameAsString(), paramSig(encl.get()));
            int site = mce.getBegin().map(p -> p.line).orElse(0);
            try {
                ResolvedMethodDeclaration r = mce.resolve();
                String owner = r.declaringType().getQualifiedName();
                if (!inProject(owner)) continue; // M0: keep CALLS in-project; library calls are noise
                StringBuilder ps = new StringBuilder();
                for (int i = 0; i < r.getNumberOfParams(); i++) { if (i > 0) ps.append(','); ps.append(r.getParam(i).getType().describe()); }
                String dstId = IdGen.method(owner, r.getName(), ps.toString());
                if (!nodes.containsKey(dstId))
                    node(dstId, "Method", r.getName(), "deterministic", "exact", "scip", null,
                            Collections.singletonMap("kind", "Method"));
                edge("CALLS", srcId, dstId, "deterministic", "exact", "scip",
                        sref(cu, encl.get().getBegin().map(p -> p.line).orElse(0)),
                        Collections.singletonMap("call_site", sref(cu, site)));
            } catch (Throwable t) { unresolved++; }
        }
    }

    // ---- run -----------------------------------------------------------------
    void run(List<String> srcRoots, String classpath, String out) throws Exception {
        CombinedTypeSolver ts = new CombinedTypeSolver(new ReflectionTypeSolver(false));
        for (String r : srcRoots) ts.add(new JavaParserTypeSolver(new File(r)));
        ts.add(new ClassLoaderTypeSolver(classpathLoader(classpath)));   // target/classes (Lombok/JAXB) + dep jars
        ParserConfiguration cfg = new ParserConfiguration()
                .setLanguageLevel(ParserConfiguration.LanguageLevel.JAVA_17)
                .setSymbolResolver(new JavaSymbolSolver(ts));
        List<CompilationUnit> cus = new ArrayList<>();
        for (String r : srcRoots) {
            SourceRoot sr = new SourceRoot(Paths.get(r), cfg);
            sr.tryToParse().forEach(pr -> pr.getResult().ifPresent(cus::add));
        }
        for (CompilationUnit cu : cus) declarations(cu);   // pass 1: every target node exists first
        for (CompilationUnit cu : cus) calls(cu);          // pass 2: resolve references
        Map<String, Object> g = new LinkedHashMap<>();
        g.put("index_version", IV); g.put("repo", repo); g.put("commit", sha);
        g.put("nodes", new ArrayList<>(nodes.values()));
        g.put("edges", new ArrayList<>(edges.values()));
        om.writerWithDefaultPrettyPrinter().writeValue(new File(out), g);
        System.out.printf("wrote %s: %d nodes, %d edges (%d unresolved calls)%n", out, nodes.size(), edges.size(), unresolved);
    }
    static URLClassLoader classpathLoader(String cp) throws Exception {
        List<URL> urls = new ArrayList<>();
        for (String e : cp.split(File.pathSeparator)) if (!e.isEmpty()) urls.add(new File(e).toURI().toURL());
        return new URLClassLoader(urls.toArray(new URL[0]), Extractor.class.getClassLoader());
    }

    public static void main(String[] args) throws Exception {
        Map<String, String> a = new HashMap<>();
        for (int i = 0; i < args.length - 1; i += 2) a.put(args[i].replaceFirst("^--", ""), args[i + 1]);
        Extractor ex = new Extractor(a.get("repo"), a.get("sha"), a.getOrDefault("basepkg", "com.att.creditcheck"));
        List<String> roots = new ArrayList<>();
        roots.add(a.get("src"));
        if (a.containsKey("gen") && !a.get("gen").isEmpty()) roots.addAll(Arrays.asList(a.get("gen").split(",")));
        ex.run(roots, a.getOrDefault("classpath", ""), a.getOrDefault("out", "graph.json"));
    }
}
