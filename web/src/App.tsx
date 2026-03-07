import { createBrowserRouter, Link, RouterProvider } from "react-router";
import { Layout } from "@/components/Layout";
import { Dashboard } from "@/pages/Dashboard";
import { BlockExplorer } from "@/pages/BlockExplorer";
import { BlockDetail } from "@/pages/BlockDetail";
import { TxDetail } from "@/pages/TxDetail";
import { Mempool } from "@/pages/Mempool";
import { Mining } from "@/pages/Mining";
import { Address } from "@/pages/Address";

function NotFound() {
  return (
    <div className="py-12 text-center">
      <h1 className="text-2xl font-bold text-zinc-100">Page not found</h1>
      <p className="mt-2 text-zinc-500">
        The page you are looking for does not exist.
      </p>
      <Link to="/" className="mt-4 inline-block text-blue-400 hover:underline">
        Back to Dashboard
      </Link>
    </div>
  );
}

const router = createBrowserRouter([
  {
    path: "/",
    element: <Layout />,
    children: [
      { index: true, element: <Dashboard /> },
      { path: "blocks", element: <BlockExplorer /> },
      { path: "blocks/:height", element: <BlockDetail /> },
      { path: "tx/:hash", element: <TxDetail /> },
      { path: "mempool", element: <Mempool /> },
      { path: "mining", element: <Mining /> },
      { path: "address/:addr", element: <Address /> },
      { path: "*", element: <NotFound /> },
    ],
  },
]);

function App() {
  return <RouterProvider router={router} />;
}

export default App;
