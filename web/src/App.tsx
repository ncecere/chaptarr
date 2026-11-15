import { QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { Toaster } from './components/Toaster'
import { AppShell } from './layout/AppShell'
import { queryClient } from './lib/queryClient'
import { ActivityPage } from './pages/ActivityPage'
import { AddPage } from './pages/AddPage'
import { AuthorDetailPage } from './pages/AuthorDetailPage'
import { AuthorsPage } from './pages/AuthorsPage'
import { AuthGate } from './pages/auth/AuthGate'
import { BookDetailPage } from './pages/BookDetailPage'
import { LibraryPage } from './pages/LibraryPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { SeriesDetailPage } from './pages/SeriesDetailPage'
import { SeriesPage } from './pages/SeriesPage'
import { AccountSettings } from './pages/settings/AccountSettings'
import { DownloadClientsSettings } from './pages/settings/DownloadClientsSettings'
import { GeneralSettings } from './pages/settings/GeneralSettings'
import { MetadataSettings } from './pages/settings/MetadataSettings'
import { IndexersSettings } from './pages/settings/IndexersSettings'
import { RootFoldersSettings } from './pages/settings/RootFoldersSettings'
import { SettingsLayout } from './pages/settings/SettingsLayout'
import { SystemPage } from './pages/SystemPage'
import { WantedPage } from './pages/WantedPage'

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthGate>
          <Routes>
            <Route element={<AppShell />}>
              <Route index element={<LibraryPage />} />
              <Route path="authors" element={<AuthorsPage />} />
              <Route path="authors/:id" element={<AuthorDetailPage />} />
              <Route path="books/:id" element={<BookDetailPage />} />
              <Route path="series" element={<SeriesPage />} />
              <Route path="series/:id" element={<SeriesDetailPage />} />
              <Route path="add" element={<AddPage />} />
              <Route path="wanted" element={<WantedPage />} />
              <Route path="activity" element={<ActivityPage />} />
              <Route path="settings" element={<SettingsLayout />}>
                <Route index element={<Navigate to="general" replace />} />
                <Route path="general" element={<GeneralSettings />} />
                <Route path="metadata" element={<MetadataSettings />} />
                <Route path="rootfolders" element={<RootFoldersSettings />} />
                <Route path="indexers" element={<IndexersSettings />} />
                <Route path="downloadclients" element={<DownloadClientsSettings />} />
                <Route path="account" element={<AccountSettings />} />
              </Route>
              <Route path="system" element={<SystemPage />} />
              <Route path="*" element={<NotFoundPage />} />
            </Route>
          </Routes>
        </AuthGate>
      </BrowserRouter>
      <Toaster />
    </QueryClientProvider>
  )
}
