import { useState, useCallback, useEffect } from "react";
import {
  Label,
  Button,
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  FormGroup,
  FormSelect,
  FormSelectOption,
  TextInput,
  Alert,
} from "@patternfly/react-core";
import { useAuth } from "../auth/AuthContext";

interface SharingData {
  asset_type: string;
  asset_name: string;
  owner: string;
  tenant: string;
  visibility: "private" | "team" | "public";
  shared_with: string[];
}

const VIS_COLORS: Record<string, "grey" | "blue" | "green"> = {
  private: "grey",
  team: "blue",
  public: "green",
};

const VIS_ICONS: Record<string, string> = {
  private: "🔒",
  team: "👥",
  public: "🌐",
};

interface ShareBadgeProps {
  assetType: string;
  assetName: string;
  compact?: boolean;
}

export const ShareBadge = ({ assetType, assetName, compact }: ShareBadgeProps) => {
  const { authHeaders, isAtLeast } = useAuth();
  const [sharing, setSharing] = useState<SharingData | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [vis, setVis] = useState<string>("private");
  const [sharedWith, setSharedWith] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const canEdit = isAtLeast("developer");

  const fetchSharing = useCallback(async () => {
    try {
      const res = await fetch(
        `/api/v1/sharing?type=${encodeURIComponent(assetType)}&name=${encodeURIComponent(assetName)}`,
        { headers: authHeaders() },
      );
      if (res.ok) {
        const data = await res.json();
        setSharing(data);
        setVis(data.visibility || "private");
        setSharedWith((data.shared_with || []).join(", "));
      }
    } catch {
      /* ignore */
    }
  }, [assetType, assetName, authHeaders]);

  useEffect(() => {
    fetchSharing();
  }, [fetchSharing]);

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      const shared = sharedWith
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean);
      const res = await fetch("/api/v1/sharing", {
        method: "PUT",
        headers: { ...authHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify({
          asset_type: assetType,
          asset_name: assetName,
          visibility: vis,
          shared_with: shared,
        }),
      });
      if (res.ok) {
        const data = await res.json();
        setSharing(data);
        setModalOpen(false);
      } else {
        const errData = await res.json();
        setError(errData.error || "Failed to update sharing");
      }
    } catch {
      setError("Network error");
    } finally {
      setSaving(false);
    }
  };

  const currentVis = sharing?.visibility || "private";

  return (
    <>
      <Label
        color={VIS_COLORS[currentVis]}
        isCompact={compact}
        onClick={canEdit ? () => setModalOpen(true) : undefined}
        style={canEdit ? { cursor: "pointer" } : undefined}
      >
        {VIS_ICONS[currentVis]} {currentVis}
        {sharing?.shared_with && sharing.shared_with.length > 0
          ? ` (${sharing.shared_with.length})`
          : ""}
      </Label>

      {modalOpen && (
        <Modal
          isOpen={modalOpen}
          onClose={() => setModalOpen(false)}
          variant="small"
          aria-label="Share asset"
        >
          <ModalHeader title={`Share: ${assetName}`} />
          <ModalBody>
            {error && (
              <Alert variant="danger" isInline title={error} style={{ marginBottom: 12 }} />
            )}
            <FormGroup label="Visibility" fieldId="share-vis" style={{ marginBottom: 16 }}>
              <FormSelect
                id="share-vis"
                value={vis}
                onChange={(_e, v) => setVis(v)}
              >
                <FormSelectOption value="private" label="🔒 Private — only you" />
                <FormSelectOption value="team" label="👥 Team — your tenant" />
                <FormSelectOption value="public" label="🌐 Public — all tenants" />
              </FormSelect>
            </FormGroup>
            {vis === "team" && (
              <FormGroup
                label="Share with specific users (comma-separated)"
                fieldId="share-users"
              >
                <TextInput
                  id="share-users"
                  value={sharedWith}
                  onChange={(_e, v) => setSharedWith(v)}
                  placeholder="e.g. maya, alex, priya"
                />
              </FormGroup>
            )}
            {sharing?.owner && (
              <div style={{ marginTop: 12, fontSize: 12, color: "var(--pf-t--global--text--color--subtle)" }}>
                Owner: <strong>{sharing.owner}</strong> · Tenant: <strong>{sharing.tenant}</strong>
              </div>
            )}
          </ModalBody>
          <ModalFooter>
            <Button onClick={handleSave} isLoading={saving} isDisabled={saving}>
              Save
            </Button>
            <Button variant="link" onClick={() => setModalOpen(false)}>
              Cancel
            </Button>
          </ModalFooter>
        </Modal>
      )}
    </>
  );
};
