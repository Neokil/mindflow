import { useEffect, useState } from "react";
import { api } from "../../api";
import { ErrorMessage } from "../../ui";
export default function ProductUpdates() {
  const [enabled, setEnabled] = useState(false),
    [loading, setLoading] = useState(true),
    [error, setError] = useState("");
  useEffect(() => {
    void api<{ enabled: boolean }>("/auth/product-updates")
      .then((result) => setEnabled(result.enabled))
      .catch((error) => setError(error.message))
      .finally(() => setLoading(false));
  }, []);
  async function change(value: boolean) {
    setLoading(true);
    setError("");
    try {
      await api("/auth/product-updates", "PATCH", { enabled: value });
      setEnabled(value);
    } catch (error) {
      setError((error as Error).message);
    } finally {
      setLoading(false);
    }
  }
  return (
    <section>
      <h3>Product updates</h3>
      <label className="consent-option">
        <input
          type="checkbox"
          checked={enabled}
          disabled={loading}
          onChange={(event) => void change(event.target.checked)}
        />
        Email me when a new mindflow version is available.
      </label>
      <ErrorMessage message={error} />
    </section>
  );
}
