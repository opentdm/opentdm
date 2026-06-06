import { Config } from "../../api";
import KvEditor from "./KvEditor";
import CodeEditor from "./CodeEditor";
import CsvEditor from "./CsvEditor";

interface EditorDispatchProps {
  slug: string;
  config: Config;
  layer: string;
  readOnly?: boolean;
}

// Picks the right editor for an object's format:
//   env/properties/secret → key/value table (KvEditor)
//   json/xml              → code editor with Format + validate (CodeEditor)
//   csv                   → code editor + table preview (CsvEditor)
// readOnly hides the save/mutation controls (viewer role).
export default function EditorDispatch({ slug, config, layer, readOnly }: EditorDispatchProps) {
  switch (config.format) {
    case "json":
    case "xml":
      return <CodeEditor slug={slug} config={config} layer={layer} readOnly={readOnly} />;
    case "csv":
      return <CsvEditor slug={slug} config={config} layer={layer} readOnly={readOnly} />;
    default:
      return <KvEditor slug={slug} config={config} layer={layer} readOnly={readOnly} />;
  }
}
