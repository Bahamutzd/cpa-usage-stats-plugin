import { ReactNode, useCallback, useLayoutEffect, useRef, useState } from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { MainRoutes } from '@/router/MainRoutes';
import { IconSidebarDashboard, IconSidebarMonitor } from '@/components/ui/icons';
import { INLINE_LOGO_JPEG } from '@/assets/logoInline';
import { useThemeStore } from '@/stores';
import { STORAGE_KEY_SIDEBAR } from '@/utils/constants';
import type { Theme } from '@/types';

const SIDEBAR_ICON_SIZE = 20;

type NavItem = { path: string; label: string; shortLabel: string; icon: ReactNode };

export function MainLayout() {
  const { t } = useTranslation();
  const location = useLocation();
  const theme = useThemeStore((state) => state.theme);
  const setTheme = useThemeStore((state) => state.setTheme);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    try { return localStorage.getItem(STORAGE_KEY_SIDEBAR) === 'true'; }
    catch { return false; }
  });
  const contentRef = useRef<HTMLDivElement | null>(null);
  const headerRef = useRef<HTMLElement | null>(null);

  const fullBrandName = 'CPA Usage Stats';
  const showSidebarLabels = !sidebarCollapsed || sidebarOpen;

  useLayoutEffect(() => {
    const update = () => {
      const h = headerRef.current?.offsetHeight;
      if (h) document.documentElement.style.setProperty('--header-height', `${h}px`);
    };
    update();
    const ro = new ResizeObserver(update);
    if (headerRef.current) ro.observe(headerRef.current);
    return () => ro.disconnect();
  }, []);

  useLayoutEffect(() => {
    const update = () => {
      const el = contentRef.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      document.documentElement.style.setProperty('--content-center-x', `${rect.left + rect.width / 2}px`);
    };
    update();
    const ro = new ResizeObserver(update);
    if (contentRef.current) ro.observe(contentRef.current);
    return () => ro.disconnect();
  }, []);

  const handleThemeToggle = useCallback(() => {
    const next: Theme = theme === 'dark' ? 'white' : 'dark';
    setTheme(next);
  }, [theme, setTheme]);

  const navSections: NavItem[][] = [[
    { path: '/', label: t('nav.dashboard'), shortLabel: t('nav.dashboard'), icon: <IconSidebarDashboard size={SIDEBAR_ICON_SIZE} /> },
    { path: '/monitoring', label: t('nav.monitoring_center'), shortLabel: t('nav.monitoring_center'), icon: <IconSidebarMonitor size={SIDEBAR_ICON_SIZE} /> },
    { path: '/model-prices', label: t('nav.model_prices', { defaultValue: 'Model Prices' }), shortLabel: t('nav.model_prices', { defaultValue: 'Prices' }), icon: <IconSidebarMonitor size={SIDEBAR_ICON_SIZE} /> },
  ]].filter(s => s.length > 0);

  const navItems = navSections.flat();
  const currentPath = location.pathname === '/dashboard' ? '/' : location.pathname;
  const matchesNav = (item: NavItem, pathname: string) => item.path === '/' ? pathname === '/' : pathname === item.path || pathname.startsWith(`${item.path}/`);
  const activeNav = [...navItems].sort((a, b) => b.path.length - a.path.length).find(item => matchesNav(item, currentPath)) ?? navItems[0];
  const currentLabel = activeNav?.label ?? fullBrandName;

  const mobileToggleLabel = sidebarOpen ? 'Close navigation' : 'Open navigation';

  return (
    <div className={['app-shell', sidebarCollapsed ? 'sidebar-is-collapsed' : ''].filter(Boolean).join(' ')}>
      <header className="main-header" ref={headerRef}>
        <div className="navbar">
          <div className="navbar-left">
            <button type="button" className="hamburger-container"
              onClick={() => {
                if (window.matchMedia('(max-width: 768px)').matches) { setSidebarOpen(p => !p); return; }
                setSidebarCollapsed(p => { const n = !p; try { localStorage.setItem(STORAGE_KEY_SIDEBAR, String(n)); } catch {} return n; });
              }}
              title={mobileToggleLabel} aria-label={mobileToggleLabel}>
              {sidebarOpen ? <CloseIcon /> : sidebarCollapsed ? <ExpandIcon /> : <CollapseIcon />}
            </button>
            <nav className="app-breadcrumb" aria-label="Navigation">
              <span className="breadcrumb-item">{currentLabel}</span>
            </nav>
          </div>
          <div className="navbar-right">
            <button type="button" onClick={handleThemeToggle} className="theme-toggle-btn" title={theme === 'dark' ? 'Light mode' : 'Dark mode'} aria-label="Toggle theme">
              {theme === 'dark' ? <SunIcon /> : <MoonIcon />}
            </button>
          </div>
        </div>
      </header>

      <div className="main-body">
        <button type="button" className={`sidebar-backdrop ${sidebarOpen ? 'visible' : ''}`}
          onClick={() => setSidebarOpen(false)} aria-label="Close" aria-hidden={!sidebarOpen} tabIndex={sidebarOpen ? 0 : -1} />

        <aside className={`sidebar ${sidebarOpen ? 'open' : ''} ${sidebarCollapsed ? 'collapsed' : ''}`}>
          <div className="sidebar-brand" title={fullBrandName}>
            <div className="sidebar-brand-main">
              <img src={INLINE_LOGO_JPEG} alt="CPA logo" className="sidebar-brand-logo" />
              {showSidebarLabels && <span className="sidebar-brand-title">CPA</span>}
            </div>
          </div>
          <div className="nav-section">
            {navSections.map((section, si) => (
              <div className="nav-menu-section" key={`nav-${si}`}>
                {section.map(item => (
                  <NavLink key={item.path} to={item.path} end={item.path === '/'}
                    className={({ isActive }) => `nav-item ${isActive || matchesNav(item, currentPath) ? 'active' : ''}`}
                    onClick={() => setSidebarOpen(false)} title={item.label}>
                    <span className="nav-icon">{item.icon}</span>
                    {showSidebarLabels && <span className="nav-label">{item.shortLabel}</span>}
                  </NavLink>
                ))}
              </div>
            ))}
          </div>
        </aside>

        <div className="content" ref={contentRef}>
          <main className="main-content">
            <MainRoutes />
          </main>
        </div>
      </div>
    </div>
  );
}

function CollapseIcon() { return <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><rect x={3} y={4} width={18} height={16} rx={2} /><path d="M9 4v16" /><path d="m16 9-3 3 3 3" /></svg>; }
function ExpandIcon() { return <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><rect x={3} y={4} width={18} height={16} rx={2} /><path d="M9 4v16" /><path d="m13 9 3 3-3 3" /></svg>; }
function CloseIcon() { return <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="M18 6 6 18" /><path d="m6 6 12 12" /></svg>; }
function SunIcon() { return <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><circle cx={12} cy={12} r={4} /><path d="M12 2v2M12 20v2m-7.07-17.07 1.41 1.41m9.32 9.32 1.41 1.41M2 12h2M20 12h2m-13.66 5.66-1.41 1.41m9.32-9.32-1.41 1.41" /></svg>; }
function MoonIcon() { return <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9z" /></svg>; }