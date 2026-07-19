import { Link, NavLink, Route, Routes } from "react-router-dom";
import { IDENTITIES, useIdentity } from "./identity";
import JobsPage from "./pages/JobsPage";
import SubmitPage from "./pages/SubmitPage";
import JobDetailPage from "./pages/JobDetailPage";
import LibraryPage from "./pages/library/LibraryPage";
import LibrarySearchPage from "./pages/library/LibrarySearchPage";

function Header() {
  const { identity, setIdentity } = useIdentity();
  return (
    <header className="app-header">
      <div className="header-inner">
        <Link to="/" className="app-title">
          Podcast Transcript Agent
        </Link>
        <nav className="app-nav">
          <NavLink to="/" end>
            Jobs
          </NavLink>
          <NavLink to="/library">Library</NavLink>
          <NavLink to="/submit">Submit</NavLink>
        </nav>
        <div className="identity-switcher">
          <label htmlFor="identity-select" className="muted">
            Acting as
          </label>
          <select
            id="identity-select"
            value={identity.userId}
            onChange={(e) => {
              const next = IDENTITIES.find((i) => i.userId === e.target.value);
              if (next) setIdentity(next);
            }}
          >
            {IDENTITIES.map((i) => (
              <option key={i.userId} value={i.userId}>
                {i.userId} ({i.role})
              </option>
            ))}
          </select>
        </div>
      </div>
    </header>
  );
}

export default function App() {
  return (
    <>
      <Header />
      <main className="app-main">
        <Routes>
          <Route path="/" element={<JobsPage />} />
          <Route path="/library" element={<LibraryPage />} />
          <Route path="/library/search" element={<LibrarySearchPage />} />
          <Route path="/submit" element={<SubmitPage />} />
          <Route path="/jobs/:jobId" element={<JobDetailPage />} />
          <Route
            path="*"
            element={
              <div className="empty-state">
                Page not found. <Link to="/">Back to jobs</Link>
              </div>
            }
          />
        </Routes>
      </main>
    </>
  );
}
