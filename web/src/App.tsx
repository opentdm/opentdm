import { useEffect, useState } from "react";
import { Routes, Route, Navigate, useNavigate, useLocation } from "react-router-dom";
import { Box, Spinner } from "./ui/primer";
import { api, User } from "./api";
import Sidebar from "./components/Sidebar";
import Topbar from "./components/Topbar";
import CommandPalette from "./components/CommandPalette";
import { ProjectsProvider } from "./lib/projects";
import { hydratePreferences } from "./lib/preferences";
import Setup from "./pages/Setup";
import Login from "./pages/Login";
import Projects from "./pages/Projects";
import ProjectPage from "./pages/Project";
import ObjectPage from "./pages/ObjectPage";
import ProjectSettings from "./pages/ProjectSettings";
import Settings from "./pages/Settings";
import Activity from "./pages/Activity";
import AcceptInvite from "./pages/AcceptInvite";

export default function App() {
  const [me, setMe] = useState<User | null | undefined>(undefined);
  const [needsSetup, setNeedsSetup] = useState(false);

  async function refresh() {
    try {
      const s = await api.get<{ needs_setup: boolean }>("/auth/setup");
      if (s.needs_setup) {
        setNeedsSetup(true);
        setMe(null);
        return;
      }
    } catch {
      /* ignore */
    }
    try {
      const user = await api.get<User>("/auth/me");
      hydratePreferences(user.preferences);
      setMe(user);
    } catch {
      setMe(null);
    }
  }

  useEffect(() => {
    void refresh();
  }, []);

  if (me === undefined) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", pt: 6 }}>
        <Spinner />
      </Box>
    );
  }

  return (
    <Routes>
      <Route
        path="/setup"
        element={
          me ? (
            <Navigate to="/" replace />
          ) : needsSetup ? (
            <Setup onDone={refresh} />
          ) : (
            <Navigate to="/login" replace />
          )
        }
      />
      <Route path="/login" element={me ? <Navigate to="/" replace /> : <Login onDone={refresh} />} />
      {/* Public: reachable whether or not signed in (accepting creates + logs in). */}
      <Route path="/accept-invite" element={<AcceptInvite onDone={refresh} />} />
      <Route
        path="/*"
        element={
          me ? (
            <Shell me={me} onLogout={refresh} />
          ) : (
            <Navigate to={needsSetup ? "/setup" : "/login"} replace />
          )
        }
      />
    </Routes>
  );
}

function Shell({ me, onLogout }: { me: User; onLogout: () => void }) {
  const nav = useNavigate();
  const { pathname } = useLocation();
  const [cmdkOpen, setCmdkOpen] = useState(false);
  async function logout() {
    await api.post("/auth/logout");
    onLogout();
    nav("/login");
  }

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setCmdkOpen((v) => !v);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  return (
    <ProjectsProvider>
      <Box className="otdm-app">
        <Sidebar me={me} onSignOut={logout} onSearch={() => setCmdkOpen(true)} />
        <Box className="otdm-content">
          <Topbar onOpenPalette={() => setCmdkOpen(true)} />
          <Box className="otdm-content-inner">
            <div className="otdm-page" key={pathname}>
              <Routes>
              <Route path="/" element={<Projects />} />
              <Route path="/projects/:slug" element={<ProjectPage />} />
              <Route path="/projects/:slug/configs/:configId" element={<ObjectPage />} />
              <Route path="/projects/:slug/settings" element={<ProjectSettings />} />
              <Route path="/projects/:slug/activity" element={<Activity />} />
              <Route path="/settings" element={<Settings me={me} />} />
              <Route path="/settings/:section" element={<Settings me={me} />} />
              {/* Back-compat for the old top-level admin routes. */}
              <Route path="/users" element={<Navigate to="/settings/users" replace />} />
              <Route path="/activity" element={<Navigate to="/settings/activity" replace />} />
              <Route path="*" element={<Navigate to="/" replace />} />
              </Routes>
            </div>
          </Box>
        </Box>
      </Box>
      <CommandPalette open={cmdkOpen} onClose={() => setCmdkOpen(false)} />
    </ProjectsProvider>
  );
}
