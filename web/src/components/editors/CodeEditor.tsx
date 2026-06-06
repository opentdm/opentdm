import { Suspense, lazy, useEffect, useState } from "react";
import { Box, Button, Flash, Spinner, Text } from "@primer/react";
import { api, Config } from "../../api";
import type { EditorLanguage } from "./CodeMirrorLazy";

const CodeMirrorLazy = lazy(() => import("./CodeMirrorLazy"));

const contentType: Record<string, string> = {
  json: "application/json",
  xml: "application/xml",
  csv: "text/csv",
};

interface CodeEditorProps {
  slug: string;
  config: Config;
  layer: string;
}

// JSON/XML object editor: a real code editor with Format + client-side
// validate-on-save. The server's codec.ValidateFile is the authority — we
// surface its detail — but blocking obviously-invalid content here avoids a
// round-trip and gives instant feedback.
export default function CodeEditor({ slug, config, layer }: CodeEditorProps) {
  const language: EditorLanguage = config.format === "json" ? "json" : "xml";
  const [text, setText] = useState("");
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");
  const [msg, setMsg] = useState("");

  useEffect(() => {
    setMsg("");
    setErr("");
    setLoading(true);
    api
      .getText(`/projects/${slug}/configs/${config.id}/blob?env=${encodeURIComponent(layer)}`)
      .then(setText)
      .catch(() => setText("")) // no content for this layer yet
      .finally(() => setLoading(false));
  }, [config.id, layer]);

  function format() {
    setErr("");
    try {
      setText(config.format === "json" ? formatJson(text) : formatXml(text));
      setMsg("Formatted.");
    } catch (e: any) {
      setErr(`Cannot format: ${e.message}`);
    }
  }

  async function save() {
    setErr("");
    setMsg("");
    const problem = validate(config.format, text);
    if (problem) {
      setErr(problem);
      return;
    }
    try {
      await api.putRaw(
        `/projects/${slug}/configs/${config.id}/blob?env=${encodeURIComponent(layer)}`,
        text,
        contentType[config.format] || "application/octet-stream",
      );
      setMsg(`Saved ${layer}.`);
    } catch (e: any) {
      setErr(e.message);
    }
  }

  if (loading) return <Spinner />;
  return (
    <Box>
      {err && (
        <Flash variant="danger" sx={{ mb: 2 }}>
          {err}
        </Flash>
      )}
      <Box sx={{ borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2, overflow: "hidden" }}>
        <Suspense fallback={<Box sx={{ p: 3 }}><Spinner size="small" /></Box>}>
          <CodeMirrorLazy value={text} onChange={setText} language={language} />
        </Suspense>
      </Box>
      <Box sx={{ mt: 2, display: "flex", gap: 2, alignItems: "center", flexWrap: "wrap" }}>
        <Button variant="primary" onClick={save}>
          Save {layer}
        </Button>
        <Button onClick={format}>Format</Button>
        {msg && <Text sx={{ color: "success.fg" }}>{msg}</Text>}
      </Box>
      <Text sx={{ color: "fg.muted", fontSize: 0, display: "block", mt: 2 }}>
        Editing the <b>{layer}</b> layer. File content fully replaces the layer on save.
      </Text>
    </Box>
  );
}

function formatJson(text: string): string {
  return JSON.stringify(JSON.parse(text), null, 2) + "\n";
}

// Light XML pretty-printer: parse to validate, then re-indent by tag depth.
// Not a full serializer — good enough for human-readable formatting of the
// config blobs we store; the server remains the validation authority.
function formatXml(text: string): string {
  const doc = new DOMParser().parseFromString(text, "application/xml");
  const errNode = doc.querySelector("parsererror");
  if (errNode) throw new Error(errNode.textContent?.split("\n")[0] || "invalid XML");
  const withBreaks = text.replace(/>\s*</g, ">\n<").trim();
  let depth = 0;
  return (
    withBreaks
      .split("\n")
      .map((line) => {
        const l = line.trim();
        if (l.startsWith("</")) depth = Math.max(0, depth - 1);
        const indent = "  ".repeat(depth);
        if (l.startsWith("<") && !l.startsWith("</") && !l.startsWith("<?") && !l.endsWith("/>") && !l.includes("</")) {
          depth += 1;
        }
        return indent + l;
      })
      .join("\n") + "\n"
  );
}

// Returns an error message if the content is invalid, or "" if it's fine.
function validate(format: string, text: string): string {
  if (!text.trim()) return "";
  if (format === "json") {
    try {
      JSON.parse(text);
    } catch (e: any) {
      return `Invalid JSON: ${e.message}`;
    }
  } else if (format === "xml") {
    const doc = new DOMParser().parseFromString(text, "application/xml");
    const errNode = doc.querySelector("parsererror");
    if (errNode) return `Invalid XML: ${errNode.textContent?.split("\n")[0] || "parse error"}`;
  }
  return "";
}
