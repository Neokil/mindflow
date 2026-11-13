type Props = {
  label: string;
  value: string;
  disabled?: boolean;
  onChange: (color: string) => void;
};

// Keep the native picker for choosing RGB; render the actual color separately
// so transparent and eight-digit document colors remain visible and accurate.
export default function ColorInput({
  label,
  value,
  disabled,
  onChange,
}: Props) {
  const rgb = /^#[0-9a-f]{6}([0-9a-f]{2})?$/i.test(value)
    ? value.slice(0, 7)
    : "#ffffff";
  return (
    <span className={`color-picker${disabled ? " disabled" : ""}`}>
      <span className="color-picker-swatch" aria-hidden="true">
        <span
          className="color-picker-color"
          style={{ backgroundColor: value }}
        />
      </span>
      <span className="color-picker-value" aria-hidden="true">
        {value === "transparent" ? "Transparent" : value.toUpperCase()}
      </span>
      <input
        type="color"
        aria-label={label}
        value={rgb}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
      />
    </span>
  );
}
