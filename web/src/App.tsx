import { useEffect, useState } from "react";
import { Routes, Route, Navigate, Link as RouterLink, useNavigate } from "react-router-dom";
import { Box, Header, Spinner, Text } from "@primer/react";
import { PackageIcon } from "@primer/octicons-react";
import { api, User } from "./api";
import Setup from "./pages/Setup";
import Login from "./pages/Login";
import Projects from "./pages/Projects";
import ProjectPage from "./pages/Project";
import ObjectPage from "./pages/ObjectPage";
import ProjectSettings from "./pages/ProjectSettings";
import Settings from "./pages/Settings";
import Users from "./pages/Users";
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
      setMe(await api.get<User>("/auth/me"));
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
  async function logout() {
    await api.post("/auth/logout");
    onLogout();
    nav("/login");
  }
  return (
    <>
      <Header>
        <Header.Item>
          <Header.Link as={RouterLink} to="/" sx={{ fontSize: 2, display: "flex", alignItems: "center" }}>
            <PackageIcon size={24} />
            <Box as="span" sx={{ ml: 2 }}>
              opentdm
            </Box>
          </Header.Link>
        </Header.Item>
        <Header.Item full />
        <Header.Item>
          {me.is_admin && (
            <>
              <Header.Link as={RouterLink} to="/activity" sx={{ mr: 3 }}>
                Activity
              </Header.Link>
              <Header.Link as={RouterLink} to="/users" sx={{ mr: 3 }}>
                Users
              </Header.Link>
            </>
          )}
          <Header.Link as={RouterLink} to="/settings" sx={{ mr: 3 }}>
            Tokens
          </Header.Link>
          <Text sx={{ color: "fg.onEmphasis", mr: 3 }}>{me.username}</Text>
          <Header.Link
            as="button"
            onClick={logout}
            sx={{ background: "transparent", border: 0, p: 0, font: "inherit", cursor: "pointer" }}
          >
            Sign out
          </Header.Link>
        </Header.Item>
      </Header>
      <Box sx={{ maxWidth: 960, mx: "auto", p: 4 }}>
        <Routes>
          <Route path="/" element={<Projects />} />
          <Route path="/projects/:slug" element={<ProjectPage />} />
          <Route path="/projects/:slug/configs/:configId" element={<ObjectPage />} />
          <Route path="/projects/:slug/settings" element={<ProjectSettings />} />
          <Route path="/projects/:slug/activity" element={<Activity />} />
          <Route path="/settings" element={<Settings />} />
          {me.is_admin && <Route path="/users" element={<Users />} />}
          {me.is_admin && <Route path="/activity" element={<Activity />} />}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Box>
    </>
  );
}
