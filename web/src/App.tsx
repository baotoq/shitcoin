import { createBrowserRouter, RouterProvider } from "react-router";
import { Layout } from "@/components/Layout";
import { Dashboard } from "@/pages/Dashboard";

const router = createBrowserRouter([
  {
    path: "/",
    element: <Layout />,
    children: [
      { index: true, element: <Dashboard /> },
      {
        path: "blocks",
        element: (
          <div className="text-zinc-400">Blocks (coming in next plan)</div>
        ),
      },
      {
        path: "blocks/:height",
        element: (
          <div className="text-zinc-400">
            Block Detail (coming in next plan)
          </div>
        ),
      },
      {
        path: "tx/:hash",
        element: (
          <div className="text-zinc-400">
            Transaction Detail (coming in next plan)
          </div>
        ),
      },
      {
        path: "mempool",
        element: (
          <div className="text-zinc-400">Mempool (coming in next plan)</div>
        ),
      },
      {
        path: "mining",
        element: (
          <div className="text-zinc-400">Mining (coming in next plan)</div>
        ),
      },
      {
        path: "address/:addr",
        element: (
          <div className="text-zinc-400">
            Address Detail (coming in next plan)
          </div>
        ),
      },
    ],
  },
]);

function App() {
  return <RouterProvider router={router} />;
}

export default App;
