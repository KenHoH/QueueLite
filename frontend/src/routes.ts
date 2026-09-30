/** Route metadata only. Access labels are future UI intent, not authorization guards. */
export const plannedRoutes = [
  { path: '/', title: 'Home', access: 'public' },
  { path: '/login', title: 'Login', access: 'public' },
  { path: '/register', title: 'Register', access: 'public' },
  { path: '/business/:businessId', title: 'Business detail', access: 'public' },
  { path: '/business/:businessId/join', title: 'Join queue', access: 'public' },
  { path: '/queue/:queueId', title: 'Queue ticket', access: 'ticket' },
  { path: '/my-queues', title: 'My queues', access: 'user' },
  { path: '/profile', title: 'Profile', access: 'user' },
  { path: '/business/create', title: 'Create business', access: 'user' },
  { path: '/dashboard', title: 'Business dashboard', access: 'business' },
  { path: '/counters', title: 'Counter management', access: 'business' },
  { path: '/counter/:counterId', title: 'Counter workspace', access: 'business' },
  { path: '/business/manage', title: 'Manage business', access: 'user' },
  { path: '/business/:businessId/settings', title: 'Business settings', access: 'business' },
  { path: '/plans', title: 'Plans & subscription', access: 'user' },
] as const;
