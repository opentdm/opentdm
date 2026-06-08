// Shared project list for the shell. The sidebar (All projects + Favourites)
// and the Projects landing both read from one fetch; creating a project calls
// reload() so both update without a second round-trip or a full page refresh.
import { createContext, useCallback, useContext, useEffect, useState, ReactNode } from "react";
import { api, Project } from "../api";

interface ProjectsCtx {
  projects: Project[];
  loading: boolean;
  error: string;
  reload: () => Promise<void>;
}

const Ctx = createContext<ProjectsCtx>({
  projects: [],
  loading: true,
  error: "",
  reload: async () => {},
});

export function ProjectsProvider({ children }: { children: ReactNode }) {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const reload = useCallback(async () => {
    try {
      setProjects(await api.get<Project[]>("/projects"));
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load projects");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  return <Ctx.Provider value={{ projects, loading, error, reload }}>{children}</Ctx.Provider>;
}

export function useProjectsCtx(): ProjectsCtx {
  return useContext(Ctx);
}
