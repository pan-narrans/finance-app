import WebApp from "@twa-dev/sdk";
import React, { useEffect } from "react";

interface CreateAccountWizardProps {
  mode: "create-parent" | "create-child";
  roots: string[];
  selectedParent: string;
  newSubAccount: string;
  onSelectParent: (parent: string) => void;
  onSubAccountChange: (subAccount: string) => void;
  onSelectFinal: (fullPath: string) => void;
}

export const CreateAccountWizard: React.FC<CreateAccountWizardProps> = ({
  mode,
  roots,
  selectedParent,
  newSubAccount,
  onSelectParent,
  onSubAccountChange,
  onSelectFinal,
}) => {
  useEffect(() => {
    // Main button behavior for account creation
    if (mode === "create-child" && newSubAccount.trim()) {
      WebApp.MainButton.setText(
        `CREATE & SELECT ${selectedParent}:${newSubAccount}`,
      );
      WebApp.MainButton.show();
    } else {
      WebApp.MainButton.hide();
    }

    const handleSubmit = () =>
      onSelectFinal(`${selectedParent}:${newSubAccount}`);
    WebApp.onEvent("mainButtonClicked", handleSubmit);
    return () => WebApp.offEvent("mainButtonClicked", handleSubmit);
  }, [mode, newSubAccount, selectedParent, onSelectFinal]);

  if (mode === "create-parent") {
    return (
      <div className="container">
        <div className="wizard-header">Select Top-Level Account</div>
        <div className="grid">
          {roots.map((root) => (
            <div
              key={root}
              className="grid-item"
              onClick={() => onSelectParent(root)}
            >
              {root}
            </div>
          ))}
        </div>
      </div>
    );
  }

  // mode === 'create-child'
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      const trimmed = newSubAccount.trim();
      if (trimmed && !trimmed.endsWith(":")) {
        onSubAccountChange(trimmed + ":");
      }
    }
  };

  return (
    <div className="container">
      <div className="wizard-header">New Sub-account for {selectedParent}</div>
      <div className="input-group">
        <input
          type="text"
          className="search-input"
          placeholder="e.g. Dining"
          value={newSubAccount}
          onChange={(e) => onSubAccountChange(e.target.value)}
          onKeyDown={handleKeyDown}
          autoFocus
        />
        <div className="preview">
          Full path:{" "}
          <code>
            {selectedParent}:{newSubAccount || "..."}
          </code>
        </div>
      </div>
    </div>
  );
};
