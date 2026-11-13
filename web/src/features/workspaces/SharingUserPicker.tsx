import Avatar from "../accounts/Avatar";
import type { DirectoryUser } from "../../api";

function searchText(value: string) {
  return value.normalize("NFKD").replace(/\p{M}/gu, "").toLocaleLowerCase();
}

export default function SharingUserPicker({
  users,
  query,
  onQuery,
  selected,
  onSelect,
  disabled,
  loading,
}: {
  users: DirectoryUser[];
  query: string;
  onQuery: (query: string) => void;
  selected: string;
  onSelect: (id: string) => void;
  disabled: boolean;
  loading: boolean;
}) {
  const matches = users.filter((user) =>
    searchText(user.displayName).includes(searchText(query.trim())),
  );
  return (
    <>
      <label>
        Search users
        <input
          type="search"
          autoComplete="off"
          placeholder="Search display names"
          value={query}
          disabled={disabled || loading}
          onChange={(event) => onQuery(event.target.value)}
        />
      </label>
      <fieldset className="sharing-user-options" disabled={disabled || loading}>
        <legend>Registered users</legend>
        {loading ? (
          <p className="muted" role="status">
            Loading users…
          </p>
        ) : matches.length ? (
          matches.map((user) => (
            <label className="sharing-user-option" key={user.id}>
              <input
                type="radio"
                name="workspace-recipient"
                value={user.id}
                checked={selected === user.id}
                onChange={() => onSelect(user.id)}
              />
              <Avatar
                id={user.id}
                name={user.displayName}
                version={user.avatarVersion}
              />
              <span>{user.displayName}</span>
            </label>
          ))
        ) : (
          <p className="muted" role="status">
            {query
              ? "No users match your search."
              : "No other active users are available."}
          </p>
        )}
      </fieldset>
    </>
  );
}
