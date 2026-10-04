import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import App from "./App";
import { AppStateProvider } from "./state/AppState";
import "./design/foundation.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
	<React.StrictMode>
		<BrowserRouter basename={import.meta.env.BASE_URL}>
			<AppStateProvider>
				<App />
			</AppStateProvider>
		</BrowserRouter>
	</React.StrictMode>,
);
