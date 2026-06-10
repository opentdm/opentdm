import { FormEvent, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Box, Button, Checkbox, Dialog, Flash, FormControl, IconButton, Text, TextInput } from "../ui/primer";
import { FileIcon, KeyIcon, TableIcon, TrashIcon, UploadIcon } from "@primer/octicons-react";
import { api, Config } from "../api";
import { errMessage } from "../lib/errors";
import { parseDotenv } from "../lib/dotenv";
import { parseProperties } from "../lib/properties";
import { useToast } from "../lib/toast";

const FILE_FORMATS = ["json", "csv", "xml", "yaml"];
const ACCEPT = ".env,.properties,.json,.csv,.xml,.yaml,.yml";
const CONTENT_TYPE: Record<string, string> = {
  json: "application/json",
  csv: "text/csv",
  xml: "application/xml",
  yaml: "application/yaml",
};

interface Upload {
  name: string;
  baseName: string;
  ext: string;
  text: string;
  size: number;
  lines: number;
}

// Create-object dialog with an optional file dropzone. .env/.properties become a
// variable bundle; json/csv/xml/yaml become a file object stored as the base
// variant; "mask all values" makes a secret bundle; no file → empty env bundle.
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
  const [secret, setSecret] = useState(false);
  const [over, setOver] = useState(false);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  function readFile(f?: File | null) {
    if (!f) return;
    let ext = (f.name.split(".").pop() || "").toLowerCase();
    if (ext === "yml") ext = "yaml";
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
      const varFormat = secret ? "secret" : "env";
      if (upload) {
        nm = nm || upload.baseName || "config";
        if (upload.ext === "env" || upload.ext === "properties") {
          const parsed = upload.ext === "properties" ? parseProperties(upload.text) : parseDotenv(upload.text);
          if (parsed.invalidKeys.length) {
            const shown = parsed.invalidKeys.slice(0, 3).join(", ");
            const extra = parsed.invalidKeys.length > 3 ? `, +${parsed.invalidKeys.length - 3} more` : "";
            throw new Error(`Unsupported key name(s) (must be A–Z, 0–9, _; no dots/dashes): ${shown}${extra}`);
          }
          const format = upload.ext === "properties" ? "properties" : varFormat;
          const items = secret ? parsed.items.map((it) => ({ ...it, is_secret: true })) : parsed.items;
          const created = await api.post<Config>(`/projects/${slug}/configs`, { kind: "variable", format, name: nm });
          if (items.length) await api.putItems(slug, created.id, "base", items, `Imported ${upload.name}`);
          finish(created.id);
          return;
        }
        if (FILE_FORMATS.includes(upload.ext)) {
          const created = await api.post<Config>(`/projects/${slug}/configs`, {
            kind: "file",
            format: upload.ext,
            name: nm,
          });
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
      const created = await api.post<Config>(`/projects/${slug}/configs`, {
        kind: "variable",
        format: varFormat,
        name: nm,
      });
      finish(created.id);
    } catch (e) {
      setErr(errMessage(e));
      setBusy(false);
    }
  }

  const isVariableUpload = upload?.ext === "env" || upload?.ext === "properties";
  const showSecretToggle = !upload || upload.ext === "env";
  const chip = upload?.ext === "csv" ? <TableIcon /> : isVariableUpload ? <KeyIcon /> : <FileIcon />;
  const uploadType = upload
    ? upload.ext === "env"
      ? "env variable bundle"
      : upload.ext === "properties"
        ? "properties bundle"
        : `${upload.ext} fixture`
    : "";

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
                  {uploadType} · {upload.size} bytes · {upload.lines} lines
                </Text>
              </Box>
              <IconButton
                icon={TrashIcon}
                aria-label="Remove file"
                variant="invisible"
                onClick={() => setUpload(null)}
              />
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
              <span className="otdm-dz-sub">.env, .properties, .json, .csv, .xml, .yaml</span>
            </button>
          )}
        </Box>

        {showSecretToggle && (
          <FormControl>
            <Checkbox checked={secret} onChange={(e) => setSecret(e.target.checked)} />
            <FormControl.Label>Mask all values as secrets</FormControl.Label>
            <FormControl.Caption>Creates a secret bundle — every value is hidden by default.</FormControl.Caption>
          </FormControl>
        )}

        {!upload && <Flash>No file? An empty variable bundle is created — add keys in the editor.</Flash>}

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
