/** Navigation metadata. App.tsx route boundaries enforce account/business access; the backend remains authoritative. */
export const plannedRoutes = [
	{ path: "/", title: "Home", access: "public" },
	{ path: "/login", title: "Login", access: "public" },
	{ path: "/register", title: "Register", access: "public" },
	{ path: "/business/:businessId", title: "Business detail", access: "public" },
	{ path: "/qr/:businessId", title: "Queue QR", access: "public" },
	{ path: "/business/:businessId/join", title: "Join queue", access: "public" },
	{ path: "/queue/:queueId", title: "Queue ticket", access: "ticket" },
	{ path: "/my-queues", title: "My queues", access: "user" },
	{ path: "/profile", title: "Profile", access: "user" },
	{ path: "/business/create", title: "Create business", access: "user" },
	{
		path: "/business/:businessId/dashboard",
		title: "Business dashboard",
		access: "business",
	},
	{
		path: "/business/:businessId/counters",
		title: "Counter management",
		access: "business",
	},
	{
		path: "/business/:businessId/members",
		title: "Business members",
		access: "business-manager",
	},
	{
		path: "/business/:businessId/plans",
		title: "Business plans",
		access: "business-manager",
	},
	{
		path: "/business/:businessId/share",
		title: "Share queue",
		access: "business-manager",
	},
	{ path: "/counters", title: "My service counters", access: "user" },
	{
		path: "/counter/:counterId",
		title: "Counter workspace",
		access: "business",
	},
	{ path: "/business/manage", title: "Manage business", access: "user" },
	{
		path: "/business/:businessId/settings",
		title: "Business settings",
		access: "business",
	},
	{ path: "/plans", title: "Plans & subscription", access: "user" },
] as const;
