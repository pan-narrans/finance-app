import WebApp from "@twa-dev/sdk";
import { useEffect, useState } from "react";
import "./index.css";
import { AccountSearch } from "./components/AccountSearch";
import { CreateAccountWizard } from "./components/CreateAccountWizard";

type Mode = "search" | "create-parent" | "create-child";

function App() {
  const [accounts, setAccounts] = useState<string[]>([]);
  const [roots, setRoots] = useState<string[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [mode, setMode] = useState<Mode>("search");
  const [selectedParent, setSelectedParent] = useState("");
  const [newSubAccount, setNewSubAccount] = useState("");

  // Get search type from URL params (source or target) or start_param
  const queryParams = new URLSearchParams(window.location.search);
  const type =
    queryParams.get("type") ||
    (WebApp.initDataUnsafe as any).start_param ||
    "target";

  useEffect(() => {
    WebApp.ready();
    WebApp.expand();

    fetch("/api/accounts")
      .then((res) => {
        if (!res.ok) {
          throw new Error("Failed to fetch accounts");
        }
        return res.json();
      })
      .then((data: { accounts: string[]; roots: string[] }) => {
        setAccounts(data.accounts);
        setRoots(data.roots);
        setLoading(false);
      })
      .catch((err) => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  useEffect(() => {
    // Back button behavior
    if (mode === "search") {
      WebApp.BackButton.hide();
    } else {
      WebApp.BackButton.show();
    }

    const handleBack = () => {
      if (mode === "create-parent") {
        setMode("search");
      }
      if (mode === "create-child") {
        setMode("create-parent");
      }
    };

    WebApp.onEvent("backButtonClicked", handleBack);
    return () => WebApp.offEvent("backButtonClicked", handleBack);
  }, [mode]);

  const handleSelect = async (account: string) => {
    WebApp.HapticFeedback.impactOccurred("medium");

    try {
      const response = await fetch("/api/select", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          initData: WebApp.initData,
          account,
          type,
        }),
      });

      if (!response.ok) {
        throw new Error("Selection failed");
      }
      WebApp.close();
    } catch (err) {
      WebApp.showAlert("Failed to save selection. Please try again.");
    }
  };

  if (loading) {
    return <div className="loading">Loading...</div>;
  }
  if (error) {
    return <div className="error">{error}</div>;
  }

  if (mode === "search") {
    return (
      <AccountSearch
        accounts={accounts}
        search={search}
        setSearch={setSearch}
        type={type}
        onSelect={handleSelect}
        onCreateNew={() => setMode("create-parent")}
      />
    );
  }

  return (
    <CreateAccountWizard
      mode={mode}
      roots={roots}
      selectedParent={selectedParent}
      newSubAccount={newSubAccount}
      onSelectParent={(parent) => {
        setSelectedParent(parent);
        setMode("create-child");
      }}
      onSubAccountChange={setNewSubAccount}
      onSelectFinal={handleSelect}
    />
  );
}

export default App;
