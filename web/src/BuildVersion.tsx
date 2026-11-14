import { useEffect, useState } from "react";
import { api } from "./api";
type Build = { release: string; releaseNotesURL: string };
export default function BuildVersion() {
  const [build, setBuild] = useState<Build>();
  useEffect(() => {
    void api<Build>("/version")
      .then(setBuild)
      .catch(() => {});
  }, []);
  if (!build) return null;
  return (
    <a
      className="build-version"
      href={build.releaseNotesURL}
      target="_blank"
      rel="noopener noreferrer"
      data-tooltip="View release notes"
      aria-label={`Version ${build.release}: view release notes`}
    >
      {build.release}
    </a>
  );
}
