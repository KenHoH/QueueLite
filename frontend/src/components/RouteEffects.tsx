import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';

const titles: Array<[RegExp, string]> = [
  [/^\/$/, 'Find a queue'], [/^\/my-queues$/, 'My queues'], [/^\/profile$/, 'Account'], [/^\/plans$/, 'Plans and subscription'],
  [/^\/business\/create$/, 'Create business'], [/^\/business\/manage$/, 'Your businesses'], [/\/dashboard$/, 'Business overview'],
  [/\/counters$/, 'Counters'], [/\/members$/, 'Business members'], [/\/settings$/, 'Business settings'], [/\/plans$/, 'Business subscription'],
  [/\/share$/, 'Share queue'], [/^\/business\//, 'Business'], [/^\/queue\//, 'Queue ticket'], [/^\/counter\//, 'Counter workspace'],
  [/^\/login$/, 'Sign in'], [/^\/register$/, 'Create account'],
];

export function RouteEffects() {
  const { pathname } = useLocation();
  useEffect(() => {
    document.title = `${titles.find(([pattern]) => pattern.test(pathname))?.[1] ?? 'Page'} · QueueLite`;
    window.scrollTo({ top: 0, behavior: 'auto' });
  }, [pathname]);
  return null;
}
