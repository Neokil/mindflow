export default function Avatar({
  id,
  email,
  version,
}: {
  id: string;
  email: string;
  version?: string;
}) {
  return (
    <span className="avatar" aria-hidden="true">
      {version ? (
        <img
          src={`/api/v1/accounts/${encodeURIComponent(id)}/avatar?v=${encodeURIComponent(version)}`}
          alt=""
        />
      ) : (
        email.slice(0, 2).toUpperCase()
      )}
    </span>
  );
}
