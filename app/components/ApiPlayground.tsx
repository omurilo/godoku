import React from "react";
import { cn } from "../lib/utils";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./ui/tabs";
import { Button } from "./ui/button";
import type { ApiPlaygroundData, ApiResponse } from "../types";

function MethodBadge({ method }: { method: string }) {
  const m = method.toUpperCase();
  return <span className={cn("gd-method", "gd-method--" + m.toLowerCase())}>{m}</span>;
}

function CodeBlock({ children, label }: { children: string; label?: string }) {
  return (
    <div className="gd-codeblock">
      <div className="gd-codeblock-bar">
        <span className="gd-dot" style={{ background: "#ff5f57" }} />
        <span className="gd-dot" style={{ background: "#febc2e" }} />
        <span className="gd-dot" style={{ background: "#28c840" }} />
        {label ? (
          <span style={{ marginLeft: "0.4rem", fontSize: "10px", textTransform: "uppercase", opacity: 0.5 }}>
            {label}
          </span>
        ) : null}
      </div>
      <pre>
        <code>{children}</code>
      </pre>
    </div>
  );
}

function ResponsePanel({ responses }: { responses: ApiResponse[] }) {
  const [active, setActive] = React.useState(responses[0]?.status ?? "");
  const current = responses.find((r) => r.status === active) ?? responses[0];

  return (
    <div>
      <div className="gd-status-row">
        {responses.map((r) => (
          <button
            key={r.status}
            type="button"
            onClick={() => setActive(r.status)}
            className={cn("gd-status", r.status === current?.status && "gd-status--active")}
          >
            {r.status}
          </button>
        ))}
      </div>
      {current?.description ? (
        <p style={{ margin: "0 0 0.5rem", fontSize: "0.78rem", opacity: 0.7 }}>{current.description}</p>
      ) : null}
      {current?.example ? <CodeBlock label="json">{current.example}</CodeBlock> : null}
    </div>
  );
}

export function ApiPlayground({ api }: { api: ApiPlaygroundData }) {
  const fullUrl = `${(api.baseUrl ?? "").replace(/\/$/, "")}${api.path}`;
  const params = api.params ?? [];

  return (
    <div className="gd-api">
      <div className="gd-api-head">
        <MethodBadge method={api.method} />
        <code className="gd-api-url gd-mono">{fullUrl}</code>
      </div>

      <div className="gd-api-body">
        {api.summary ? <p className="gd-api-summary">{api.summary}</p> : null}

        <Tabs defaultValue="request">
          <TabsList>
            <TabsTrigger value="request">Request</TabsTrigger>
            <TabsTrigger value="response">Response</TabsTrigger>
          </TabsList>

          <TabsContent value="request">
            {params.length ? (
              <div style={{ marginBottom: "0.75rem" }}>
                <h4 className="gd-subhead">Parameters</h4>
                <ul style={{ listStyle: "none", margin: 0, padding: 0 }}>
                  {params.map((p) => (
                    <li key={`${p.in}-${p.name}`} className="gd-param">
                      <code className="gd-param-name">{p.name}</code>
                      <span className="gd-param-in">{p.in}</span>
                      {p.type ? <span style={{ opacity: 0.7 }}>{p.type}</span> : null}
                      {p.required ? <span style={{ color: "hsl(0 70% 55%)" }}>required</span> : null}
                    </li>
                  ))}
                </ul>
              </div>
            ) : null}

            {api.requestExample ? (
              <div>
                <h4 className="gd-subhead">Body</h4>
                <CodeBlock>{api.requestExample}</CodeBlock>
              </div>
            ) : null}

            {/* <Button block disabled style={{ marginTop: "0.75rem" }}>
              Send request
            </Button> */}
          </TabsContent>

          <TabsContent value="response">
            {(api.responses ?? []).length ? (
              <ResponsePanel responses={api.responses ?? []} />
            ) : (
              <p style={{ fontSize: "0.78rem", opacity: 0.7 }}>No documented responses.</p>
            )}
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
