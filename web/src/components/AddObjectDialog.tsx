import { FormEvent, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Box, Button, Dialog, Flash, FormControl, IconButton, Text, TextInput } from "../ui/primer";
import { FileIcon, KeyIcon, TableIcon, TrashIcon, UploadIcon } from "@primer/octicons-react";
import { api, Config } from "../api";
import { parseDotenv } from "../lib/dotenv";
import { useToast } from "../lib/toast";

const FILE_FORMATS = ["json", "csv", "xml"];
const ACCEPT = ".env,.json,.csv,.xml";
const CONTENT_TYPE: Record<string, string> = {
  json: "application/json",
  csv: "text/csv",
  xml: "application/xml",
};

interface Upload {
  name: string;
  baseName: string;
  ext: string;
  text: string;
  size: number;
  lines: number;
}

// Create-object dialog with an optional file dropzone. A .env upload becomes a
// base-layer variable bundle; a json/csv/xml upload becomes a file object with
// its content stored as the base variant; no file creates an empty env bundle.
export default function AddObjectDialog({
  slug,
  onClose,
  onChange,
}: {
  slug: string;
  onClose: () => void;
  onChange: () => void;
}) {
  const nav = useNavigate();
  const toast = useToast();
  const [name, setName] = useState("");
  const [upload, setUpload] = useState<Upload | null>(null);
  const [over, setOver] = useState(false);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  function readFile(f?: File | null) {
    if (!f) return;
    const ext = (f.name.split(".").pop() || "").toLowerCase();
    const reader = new FileReader();
    reader.onload = () => {
      const text = String(reader.result || "");
      setUpload({
        name: f.name,
        baseName: f.name.replace(/\.[^.]+$/, ""),
        ext,
        text,
        size: f.size,
        lines: text.split(/\r?\n/).length,
      });
    };
    reader.readAsText(f);
  }

  function finish(id: string) {
    onChange();
    onClose();
    toast("Object created.");
    nav(`/projects/${slug}/configs/${id}`);
  }

  async function create(e: FormEvent) {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      let nm = name.trim();
      if (upload) {
        nm = nm || upload.baseName || "config";
        if (upload.ext === "env") {
          const { items, invalidKeys } = parseDotenv(upload.text);
          if (invalidKeys.length) {
            const shown = invalidKeys.slice(0, 3).join(", ");
            throw new Error(`Invalid variable name(s): ${shown}${invalidKeys.length > 3 ? "…" : ""}`);
          }
          const created = await api.post<Config>(`/projects/${slug}/configs`, { kind: "variable", format: "env", name: nm });
          if (items.length) await api.putItems(slug, created.id, "base", items, `Imported ${upload.name}`);
          finish(created.id);
          return;
        }
        if (FILE_FORMATS.includes(upload.ext)) {
          const created = await api.post<Config>(`/projects/${slug}/configs`, { kind: "file", format: upload.ext, name: nm });
          await api.putRaw(
            `/projects/${slug}/configs/${created.id}/blob?env=base`,
            upload.text,
            CONTENT_TYPE[upload.ext] || "application/octet-stream",
          );
          finish(created.id);
          return;
        }
        throw new Error(`Unsupported file type: .${upload.ext}`);
      }
      nm = nm || "new-object";
      const created = await api.post<Config>(`/projects/${slug}/configs`, { kind: "variable", format: "env", name: nm });
      finish(created.id);
    } catch (e: any) {
      setErr(e.message);
      setBusy(false);
    }
  }

  const chip = upload?.ext === "csv" ? <TableIcon /> : upload?.ext === "env" ? <KeyIcon /> : <FileIcon />;

  return (
    <Dialog title="Add object" onClose={onClose}>
      <Box as="form" onSubmit={create} sx={{ display: "grid", gap: 3 }}>
        {err && <Flash variant="danger">{err}</Flash>}
        <FormControl>
          <FormControl.Label>Name</FormControl.Label>
          <TextInput
            block
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={upload ? upload.baseName : "payments-api"}
            autoFocus
          />
          <FormControl.Caption>
            {upload ? `Imported from ${upload.name}.` : "Leave blank to use the file name when uploading."}
          </FormControl.Caption>
        </FormControl>

        <Box>
          <Text sx={{ fontWeight: "bold", fontSize: 1, display: "block", mb: 1 }}>
            Upload a file <Text sx={{ fontWeight: "normal", color: "fg.muted" }}>(optional)</Text>
          </Text>
          <input
            ref={fileRef}
            type="file"
            accept={ACCEPT}
            hidden
            onChange={(e) => {
              readFile(e.target.files?.[0]);
              e.target.value = "";
            }}
          />
          {upload ? (
            <Box className="otdm-dz-file">
              <span className="otdm-dz-ico">{chip}</span>
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Text sx={{ fontWeight: "bold", display: "block" }}>{upload.name}</Text>
                <Text sx={{ color: "fg.muted", fontSize: 0 }}>
                  {upload.ext === "env" ? "env variable bundle" : `${upload.ext} fixture`} · {upload.size} bytes ·{" "}
                  {upload.lines} lines
                </Text>
              </Box>
              <IconButton icon={TrashIcon} aria-label="Remove file" variant="invisible" onClick={() => setUpload(null)} />
            </Box>
          ) : (
            <button
              type="button"
              className={`otdm-dropzone ${over ? "over" : ""}`}
              onClick={() => fileRef.current?.click()}
              onDragOver={(e) => {
                e.preventDefault();
                setOver(true);
              }}
              onDragLeave={() => setOver(false)}
              onDrop={(e) => {
                e.preventDefault();
                setOver(false);
                readFile(e.dataTransfer.files?.[0]);
              }}
            >
              <span className="otdm-dz-up">
                <UploadIcon size={20} />
              </span>
              <span className="otdm-dz-main">Drag a file here, or click to browse</span>
              <span className="otdm-dz-sub">.env, .json, .csv, .xml</span>
            </button>
          )}
        </Box>

        {!upload && (
          <Flash>No file? An empty env variable bundle is created — add keys in the editor.</Flash>
        )}

        <Box sx={{ display: "flex", gap: 2, justifyContent: "flex-end" }}>
          <Button type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" disabled={busy}>
            {busy ? "Adding…" : "Add object"}
          </Button>
        </Box>
      </Box>
    </Dialog>
  );
}
