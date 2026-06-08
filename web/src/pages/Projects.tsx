import { FormEvent, useEffect, useState } from "react";
import { Link as RouterLink } from "react-router-dom";
import { Box, Button, FormControl, Flash, Heading, Link, Text, TextInput, sxToStyle } from "../ui/primer";
import { api, Project } from "../api";

export default function Projects() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [name, setName] = useState("");
  const [err, setErr] = useState("");
  const [showNew, setShowNew] = useState(false);

  async function load() {
    try {
      setProjects(await api.get<Project[]>("/projects"));
    } catch (e: any) {
      setErr(e.message);
    }
  }
  useEffect(() => {
    void load();
  }, []);

  async function create(e: FormEvent) {
    e.preventDefault();
    setErr("");
    try {
      await api.post("/projects", { name });
      setName("");
      setShowNew(false);
      await load();
    } catch (e: any) {
      setErr(e.message);
    }
  }

  return (
    <Box>
      <Box sx={{ display: "flex", alignItems: "center", mb: 3 }}>
        <Heading sx={{ fontSize: 3, flex: 1 }}>Projects</Heading>
        <Button variant="primary" onClick={() => setShowNew((v) => !v)}>
          New project
        </Button>
      </Box>
      {err && (
        <Flash variant="danger" sx={{ mb: 3 }}>
          {err}
        </Flash>
      )}
      {showNew && (
        <Box
          as="form"
          onSubmit={create}
          sx={{ display: "grid", gap: 2, p: 3, mb: 3, borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2 }}
        >
          <FormControl>
            <FormControl.Label>Name</FormControl.Label>
            <TextInput block value={name} onChange={(e) => setName(e.target.value)} placeholder="Payments" autoFocus />
            <FormControl.Caption>The URL slug is derived from the name.</FormControl.Caption>
          </FormControl>
          <Box>
            <Button type="submit" variant="primary">
              Create
            </Button>
          </Box>
        </Box>
      )}
      <Box sx={{ borderWidth: 1, borderStyle: "solid", borderColor: "border.default", borderRadius: 2 }}>
        {projects.length === 0 && (
          <Box sx={{ p: 4, textAlign: "center", color: "fg.muted" }}>No projects yet.</Box>
        )}
        {projects.map((p) => (
          <Box
            key={p.id}
            sx={{ p: 3, borderBottomWidth: 1, borderBottomStyle: "solid", borderColor: "border.muted" }}
          >
            <Link as={RouterLink} to={`/projects/${p.slug}`} style={sxToStyle({ fontWeight: "bold" })}>
              {p.name}
            </Link>
            <Text sx={{ color: "fg.muted", ml: 2 }}>{p.slug}</Text>
          </Box>
        ))}
      </Box>
    </Box>
  );
}
