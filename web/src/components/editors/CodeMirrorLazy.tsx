import CodeMirror from "@uiw/react-codemirror";
import { json } from "@codemirror/lang-json";
import { xml } from "@codemirror/lang-xml";
import { githubLight, githubDark } from "@uiw/codemirror-theme-github";
import { useTheme } from "../../ui/primer";

export type EditorLanguage = "json" | "xml" | "text";

interface CodeMirrorLazyProps {
  value: string;
  onChange?: (value: string) => void;
  language?: EditorLanguage;
  readOnly?: boolean;
  height?: string;
}

// Default export so it can be React.lazy()-imported; this pulls CodeMirror + the
// language extensions into a separate chunk, off the initial bundle.
export default function CodeMirrorLazy({ value, onChange, language = "text", readOnly, height }: CodeMirrorLazyProps) {
  const theme = useTheme();
  const dark = (theme.resolvedColorMode ?? theme.colorMode) === "night";
  const extensions = language === "json" ? [json()] : language === "xml" ? [xml()] : [];
  return (
    <CodeMirror
      value={value}
      onChange={onChange}
      extensions={extensions}
      theme={dark ? githubDark : githubLight}
      readOnly={readOnly}
      editable={!readOnly}
      height={height ?? "340px"}
      basicSetup={{ lineNumbers: true, foldGutter: true, highlightActiveLine: !readOnly }}
    />
  );
}
