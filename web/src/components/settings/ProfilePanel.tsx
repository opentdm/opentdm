import { Box, FormControl, Heading, Label, Text, TextInput } from "../../ui/primer";
import { User } from "../../api";

function initials(name: string): string {
  const parts = name.split(/[.\-_@\s]+/).filter(Boolean);
  const letters = parts.length >= 2 ? parts[0][0] + parts[1][0] : name.slice(0, 2);
  return letters.toUpperCase();
}

export default function ProfilePanel({ me }: { me: User }) {
  return (
    <Box>
      <Heading sx={{ fontSize: 3, mb: 1 }}>Profile</Heading>
      <Text sx={{ color: "fg.muted", display: "block", mb: 3 }}>
        Your account identity. Username and email are managed by an instance admin.
      </Text>
      <Box sx={{ display: "flex", gap: 3, alignItems: "center", mb: 4 }}>
        <span className="otdm-avatar-lg" aria-hidden="true">
          {initials(me.username)}
        </span>
        <Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
            <Text sx={{ fontWeight: "bold", fontSize: 2 }}>{me.username}</Text>
            {me.is_admin && <Label variant="accent">admin</Label>}
          </Box>
          <Text sx={{ color: "fg.muted", display: "block" }}>{me.email}</Text>
        </Box>
      </Box>
      <Box sx={{ display: "grid", gap: 3, maxWidth: 420 }}>
        <FormControl disabled>
          <FormControl.Label>Username</FormControl.Label>
          <TextInput block value={me.username} disabled />
        </FormControl>
        <FormControl disabled>
          <FormControl.Label>Email</FormControl.Label>
          <TextInput block value={me.email} disabled />
        </FormControl>
      </Box>
    </Box>
  );
}
