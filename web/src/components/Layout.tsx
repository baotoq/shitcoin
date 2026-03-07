import { NavLink, Outlet } from "react-router";
import { LayoutDashboard, Blocks, Clock, Pickaxe } from "lucide-react";
import { StatusBar } from "@/components/StatusBar";
import { SearchBar } from "@/components/SearchBar";

const navItems = [
  { to: "/", icon: LayoutDashboard, label: "Dashboard" },
  { to: "/blocks", icon: Blocks, label: "Blocks" },
  { to: "/mempool", icon: Clock, label: "Mempool" },
  { to: "/mining", icon: Pickaxe, label: "Mining" },
];

export function Layout() {
  return (
    <div className="dark flex h-screen flex-col bg-zinc-950 text-zinc-100">
      <StatusBar />
      <SearchBar />

      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        <nav className="flex w-56 flex-col gap-1 border-r border-zinc-800 bg-zinc-950 p-3">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === "/"}
              className={({ isActive }) =>
                `flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors ${
                  isActive
                    ? "bg-zinc-800 text-white"
                    : "text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200"
                }`
              }
            >
              <item.icon className="h-4 w-4" />
              {item.label}
            </NavLink>
          ))}
        </nav>

        {/* Main content */}
        <main className="flex-1 overflow-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
