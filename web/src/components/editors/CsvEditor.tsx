import { Suspense, lazy, useEffect, useMemo, useState } from "react";
import { Box, Button, Flash, Spinner, Text } from "../../ui/primer";
import { api, Config } from "../../api";
import { useToast } from "../../lib/toast";

const CodeMirrorLazy = lazy(() => import("./CodeMirrorLazy"));

interface CsvEditorProps {
  slug: string;
  config: Config;
  layer: string;
  readOnly?: boolean;
}

// CSV object editor: an editable code view (source of truth) plus a read-only
// parsed table preview so the shape is obvious at a glance.
export default function CsvEditor({ slug, config, layer, readOnly }: CsvEditorProps) {
  const toast = useToast();
  const [text, setText] = useState("");
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");

  useEffect(() => {
    setErr("");
    setLoading(true);
    api
      .getText(`/projects/${slug}/configs/${config.id}/blob?env=${encodeURIComponent(layer)}`)
      .then(setText)
      .catch(() => setText(""))
      .finally(() => setLoading(false));
  }, [slug, config.id, layer]);

  const rows = useMemo(() => parseCsv(text), [text]);

  async function save() {
    setErr("");
    try {
      await api.putRaw(`/projects/${slug}/configs/${config.id}/blob?env=${encodeURIComponent(layer)}`, text, "text/csv");
      toast(`Saved ${layer}.`);
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
          <CodeMirrorLazy value={text} onChange={setText} language="text" height="240px" readOnly={readOnly} />
        </Suspense>
      </Box>
      {!readOnly && (
        <Box sx={{ mt: 2, display: "flex", gap: 2, alignItems: "center" }}>
          <Button variant="primary" onClick={save}>
            Save {layer}
          </Button>
        </Box>
      )}
      {rows.length > 0 && (
        <Box sx={{ mt: 3 }}>
          <Text sx={{ fontWeight: "bold", display: "block", mb: 1 }}>Preview</Text>
          <Box sx={{ overflow: "auto", borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2 }}>
            <Box as="table" sx={{ borderCollapse: "collapse", width: "100%", fontSize: 0 }}>
              <Box as="thead">
                <Box as="tr">
                  {rows[0].map((cell, i) => (
                    <Box
                      as="th"
                      key={i}
                      sx={{ textAlign: "left", p: 2, bg: "canvas.subtle", borderBottomWidth: 1, borderBottomStyle: "solid", borderColor: "border.default", fontFamily: "mono" }}
                    >
                      {cell}
                    </Box>
                  ))}
                </Box>
              </Box>
              <Box as="tbody">
                {rows.slice(1).map((row, r) => (
                  <Box as="tr" key={r}>
                    {row.map((cell, c) => (
                      <Box
                        as="td"
                        key={c}
                        sx={{ p: 2, borderBottomWidth: 1, borderBottomStyle: "solid", borderColor: "border.muted", fontFamily: "mono" }}
                      >
                        {cell}
                      </Box>
                    ))}
                  </Box>
                ))}
              </Box>
            </Box>
          </Box>
          <Text sx={{ color: "fg.muted", fontSize: 0, display: "block", mt: 1 }}>
            First row shown as header · preview is read-only · edit in the code view above.
          </Text>
        </Box>
      )}
    </Box>
  );
}

// Minimal CSV parse for preview only (handles quoted fields with embedded
// commas and doubled quotes). The stored text remains the source of truth.
function parseCsv(text: string): string[][] {
  const lines = text.replace(/\r\n/g, "\n").split("\n").filter((l) => l.length > 0);
  return lines.map((line) => {
    const cells: string[] = [];
    let cur = "";
    let inQuotes = false;
    for (let i = 0; i < line.length; i++) {
      const ch = line[i];
      if (inQuotes) {
        if (ch === '"') {
          if (line[i + 1] === '"') {
            cur += '"';
            i++;
          } else {
            inQuotes = false;
          }
        } else {
          cur += ch;
        }
      } else if (ch === '"') {
        inQuotes = true;
      } else if (ch === ",") {
        cells.push(cur);
        cur = "";
      } else {
        cur += ch;
      }
    }
    cells.push(cur);
    return cells;
  });
}
