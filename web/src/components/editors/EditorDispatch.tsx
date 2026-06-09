import { Config } from "../../api";
import KvEditor from "./KvEditor";
import CodeEditor from "./CodeEditor";
import CsvEditor from "./CsvEditor";

interface EditorDispatchProps {
  slug: string;
  config: Config;
  layer: string;
  readOnly?: boolean;
  onSaved?: () => void;
}

// Picks the right editor for an object's format:
//   env/properties/secret → key/value table (KvEditor)
//   json/xml/yaml         → code editor (Format for json/xml; validate on save)
//   csv                   → code editor + table preview (CsvEditor)
// readOnly hides the save/mutation controls (viewer role). onSaved fires after a
// successful variable save so siblings (e.g. the resolved view) can refresh.
export default function EditorDispatch({ slug, config, layer, readOnly, onSaved }: EditorDispatchProps) {
  switch (config.format) {
    case "json":
    case "xml":
    case "yaml":
      return <CodeEditor slug={slug} config={config} layer={layer} readOnly={readOnly} />;
    case "csv":
      return <CsvEditor slug={slug} config={config} layer={layer} readOnly={readOnly} />;
    default:
      return <KvEditor slug={slug} config={config} layer={layer} readOnly={readOnly} onSaved={onSaved} />;
  }
}
