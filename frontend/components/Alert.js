// Messaggio di Bootstrap Italia. Gli errori usano role="alert", così i
// lettori di schermo li annunciano appena compaiono.
export default function Alert({ type = "info", children, className = "" }) {
  if (!children) {
    return null;
  }

  return (
    <div className={`alert alert-${type} ${className}`} role={type === "danger" ? "alert" : "status"}>
      {children}
    </div>
  );
}
