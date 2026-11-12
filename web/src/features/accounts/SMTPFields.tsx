import type { Settings } from "../../api";
export default function SMTPFields({
  value,
  onChange,
}: {
  value: Settings;
  onChange: (s: Settings) => void;
}) {
  const update = (key: string, v: string | number) =>
    onChange({ ...value, smtp: { ...value.smtp, [key]: v } });
  return (
    <>
      <label>
        Public application URL
        <input
          type="url"
          required
          value={value.publicURL}
          onChange={(e) => onChange({ ...value, publicURL: e.target.value })}
          placeholder="https://mindflow.example.com"
        />
      </label>
      <div className="form-row">
        <label>
          SMTP host
          <input
            required
            value={value.smtp.host}
            onChange={(e) => update("host", e.target.value)}
            placeholder="smtp.example.com"
          />
        </label>
        <label className="short">
          Port
          <input
            type="number"
            min="1"
            max="65535"
            required
            value={value.smtp.port}
            onChange={(e) => update("port", +e.target.value)}
          />
        </label>
      </div>
      <label>
        Connection security
        <select
          value={value.smtp.mode}
          onChange={(e) => update("mode", e.target.value)}
        >
          <option value="starttls">STARTTLS</option>
          <option value="tls">Implicit TLS</option>
        </select>
      </label>
      <div className="form-row">
        <label>
          SMTP username
          <input
            autoComplete="off"
            value={value.smtp.username}
            onChange={(e) => update("username", e.target.value)}
          />
        </label>
        <label>
          SMTP password
          <input
            type="password"
            autoComplete="new-password"
            value={value.smtp.password || ""}
            onChange={(e) => update("password", e.target.value)}
            placeholder="Leave blank to keep existing"
          />
        </label>
      </div>
      <div className="form-row">
        <label>
          Sender email
          <input
            type="email"
            required
            value={value.smtp.from}
            onChange={(e) => update("from", e.target.value)}
          />
        </label>
        <label>
          Sender name
          <input
            value={value.smtp.name}
            onChange={(e) => update("name", e.target.value)}
          />
        </label>
      </div>
    </>
  );
}
