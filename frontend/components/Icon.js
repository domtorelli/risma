// Icona di Bootstrap Italia dallo sprite SVG. Le icone sono decorative
// (aria-hidden): il significato deve stare nel testo accanto o in un'etichetta.
export default function Icon({ name, size, color, className = "" }) {
  const classes = ["icon", size && `icon-${size}`, color && `icon-${color}`, className].filter(Boolean).join(" ");

  return (
    <svg className={classes} aria-hidden="true" focusable="false">
      <use href={`/bootstrap-italia/svg/sprites.svg#${name}`} />
    </svg>
  );
}
