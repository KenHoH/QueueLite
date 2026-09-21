import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import App from "./App";
import { QueueProvider } from "./state/QueueContext";
import "./styles.css";
ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter>
      <QueueProvider>
        <App />
      </QueueProvider>
    </BrowserRouter>
  </React.StrictMode>,
);
