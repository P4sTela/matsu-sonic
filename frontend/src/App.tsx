import { Routes, Route, NavLink } from "react-router";
import {
  RefreshCw,
  Files,
  History,
  Send,
  Settings,
} from "lucide-react";
import { SyncPage } from "@/pages/SyncPage";
import { FilesPage } from "@/pages/FilesPage";
import { RevisionsPage } from "@/pages/RevisionsPage";
import { DistributePage } from "@/pages/DistributePage";
import { SettingsPage } from "@/pages/SettingsPage";
import { SyncProvider } from "@/hooks/SyncProvider";

const navItems = [
  { to: "/", icon: RefreshCw, label: "Sync" },
  { to: "/files", icon: Files, label: "Files" },
  { to: "/revisions", icon: History, label: "Revisions" },
  { to: "/distribute", icon: Send, label: "Distribute" },
  { to: "/settings", icon: Settings, label: "Settings" },
];

function App() {
  return (
    <SyncProvider>
      <div className="min-h-screen bg-background">
        <header className="border-b">
          <div className="max-w-5xl mx-auto px-4 flex items-center h-14 gap-6">
            <h1 className="font-bold text-lg whitespace-nowrap text-blue-700">
              Matsusonic
            </h1>
            <nav className="flex gap-1 overflow-x-auto">
              {navItems.map(({ to, icon: Icon, label }) => (
                <NavLink
                  key={to}
                  to={to}
                  end={to === "/"}
                  className={({ isActive }) =>
                    `flex items-center gap-2 px-3 py-2 rounded-md text-sm transition-colors ${
                      isActive
                        ? "bg-accent text-accent-foreground font-medium"
                        : "text-muted-foreground hover:text-foreground hover:bg-accent/50"
                    }`
                  }
                >
                  <Icon className="h-4 w-4" />
                  {label}
                </NavLink>
              ))}
            </nav>
          </div>
        </header>

        <main className="max-w-5xl mx-auto px-4 py-6">
          <Routes>
            <Route path="/" element={<SyncPage />} />
            <Route path="/files" element={<FilesPage />} />
            <Route path="/revisions" element={<RevisionsPage />} />
            <Route path="/distribute" element={<DistributePage />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Routes>
        </main>
      </div>
    </SyncProvider>
  );
}

export default App;
