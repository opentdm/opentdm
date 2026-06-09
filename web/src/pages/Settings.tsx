import { Navigate, Link as RouterLink, useParams } from "react-router-dom";
import { Box, Heading, NavList } from "../ui/primer";
import { KeyIcon, PaintbrushIcon, PeopleIcon, PersonIcon, PulseIcon } from "@primer/octicons-react";
import { User } from "../api";
import ProfilePanel from "../components/settings/ProfilePanel";
import AccessTokensPanel from "../components/settings/AccessTokensPanel";
import AppearancePanel from "../components/settings/AppearancePanel";
import ActivityPanel from "../components/settings/ActivityPanel";
import UsersPanel from "../components/settings/UsersPanel";
import Overline from "../components/Overline";

const SECTIONS = ["account", "tokens", "appearance", "activity", "users"] as const;
type Section = (typeof SECTIONS)[number];
const ADMIN_SECTIONS: Section[] = ["activity", "users"];

// GitHub-style account settings: a sticky left sub-nav + the selected panel.
// The active section comes from the /settings/:section route param.
export default function Settings({ me }: { me: User }) {
  const { section } = useParams();
  const current: Section = (SECTIONS as readonly string[]).includes(section ?? "")
    ? (section as Section)
    : "account";

  // Default-deny admin sections (UX gate; the API enforces too).
  if (ADMIN_SECTIONS.includes(current) && !me.is_admin) {
    return <Navigate to="/settings/account" replace />;
  }

  const item = (to: Section, label: string, Icon: typeof PersonIcon) => (
    <NavList.Item as={RouterLink} to={`/settings/${to}`} aria-current={current === to ? "page" : undefined}>
      <NavList.LeadingVisual>
        <Icon />
      </NavList.LeadingVisual>
      {label}
    </NavList.Item>
  );

  return (
    <Box>
      <Overline>Settings</Overline>
      <Heading sx={{ fontSize: 5 }}>Your account</Heading>
      <Box className="otdm-settings">
        <Box className="otdm-settings-nav">
          <NavList>
            <NavList.Group title="Account">
              {item("account", "Profile", PersonIcon)}
              {item("tokens", "Access tokens", KeyIcon)}
              {item("appearance", "Appearance", PaintbrushIcon)}
            </NavList.Group>
            {me.is_admin && (
              <NavList.Group title="Instance admin">
                {item("activity", "Activity", PulseIcon)}
                {item("users", "Users", PeopleIcon)}
              </NavList.Group>
            )}
          </NavList>
        </Box>
        <Box className="otdm-settings-content">
          {current === "account" && <ProfilePanel me={me} />}
          {current === "tokens" && <AccessTokensPanel />}
          {current === "appearance" && <AppearancePanel />}
          {current === "activity" && <ActivityPanel />}
          {current === "users" && <UsersPanel meId={me.id} />}
        </Box>
      </Box>
    </Box>
  );
}
