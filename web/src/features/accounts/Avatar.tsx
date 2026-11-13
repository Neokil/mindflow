export default function Avatar({
  id,
  email,
  name,
  version,
}: {
  id: string;
  email?: string;
  name?: string;
  version?: string;
}) {
  const words =
    (name || email?.split("@")[0] || "?").match(/[\p{L}\p{N}]+/gu) || [];
  const first = Array.from(words[0] || "?");
  const second = Array.from(words[1] || "");
  const initials = second[0]
    ? first[0] + second[0]
    : first.slice(0, 2).join("");
  return (
    <span className="avatar" aria-hidden="true">
      {version ? (
        <img
          src={`/api/v1/accounts/${encodeURIComponent(id)}/avatar?v=${encodeURIComponent(version)}`}
          alt=""
        />
      ) : (
        initials.toUpperCase()
      )}
    </span>
  );
}
