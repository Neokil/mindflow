import React from "react";
import { createRoot } from "react-dom/client";
import Tooltips from "./Tooltips";
import App from "./App";
import "./style.css";
createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
    <Tooltips />
  </React.StrictMode>,
);
