import { useId, useState } from "react";
import { suggestPassphrase } from "./passphrase";
export default function NewPassword({
  value,
  onChange,
  disabled,
}: {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}) {
  const [visible, setVisible] = useState(false);
  const help = useId();
  return (
    <div>
      <label>
        Password
        <input
          type={visible ? "text" : "password"}
          required
          minLength={8}
          maxLength={128}
          autoComplete="new-password"
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder="At least 8 characters"
          disabled={disabled}
          aria-describedby={help}
        />
      </label>
      <div className="password-actions">
        <button
          type="button"
          className="text-button"
          disabled={disabled}
          onClick={() => {
            onChange(suggestPassphrase());
            setVisible(true);
          }}
        >
          Suggest good password
        </button>
        <button
          type="button"
          className="text-button"
          disabled={disabled}
          onClick={() => setVisible(!visible)}
        >
          {visible ? "Hide password" : "Show password"}
        </button>
      </div>
      <p id={help} className="muted password-help">
        Random word choices add unpredictability while keeping the password
        easier to remember. Keep it unique to mindflow and save it in your
        password manager.
      </p>
    </div>
  );
}
