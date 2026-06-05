package crypto

import "encoding/binary"

// AAD builds a canonical, length-prefixed Additional Authenticated Data blob
// from its parts. Length-prefixing avoids delimiter-injection ambiguity (so
// that ("ab","c") and ("a","bc") produce distinct AAD). The result is bound
// into every Seal/Open and authenticates — but does not encrypt — the parts.
func AAD(parts ...string) []byte {
	n := 0
	for _, p := range parts {
		n += 4 + len(p)
	}
	out := make([]byte, 0, n)
	var lenBuf [4]byte
	for _, p := range parts {
		binary.BigEndian.PutUint32(lenBuf[:], uint32(len(p)))
		out = append(out, lenBuf[:]...)
		out = append(out, p...)
	}
	return out
}

// ItemAAD binds a variable value to its immutable logical identity. Note
// dek_version is intentionally NOT included: it is mutated by DEK rotation, and
// using the wrong DEK already fails GCM authentication.
func ItemAAD(projectID, envID, configID, key string) []byte {
	return AAD("item", projectID, baseOr(envID), configID, key)
}

// BlobAAD binds a file/fixture blob to its immutable logical identity.
func BlobAAD(projectID, envID, configID string) []byte {
	return AAD("blob", projectID, baseOr(envID), configID)
}

// baseOr maps the empty (project-level / base) environment to the literal
// "base" so the AAD is unambiguous and stable.
func baseOr(envID string) string {
	if envID == "" {
		return "base"
	}
	return envID
}
