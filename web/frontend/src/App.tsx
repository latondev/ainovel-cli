import { Navigate, RouterProvider, createBrowserRouter } from 'react-router-dom'
import { Layout } from './components/Layout'
import { NovelListPage } from './pages/NovelListPage'
import { NovelDetailPage } from './pages/NovelDetailPage'

const router = createBrowserRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      { index: true, element: <Navigate to="/novels" replace /> },
      { path: 'novels', element: <NovelListPage /> },
      { path: 'novels/:slug', element: <NovelDetailPage /> },
    ],
  },
])

export default function App() {
  return <RouterProvider router={router} />
}