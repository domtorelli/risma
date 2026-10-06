// Campi dei form con il markup di Bootstrap Italia, da usare con
// react-hook-form: registration è il risultato di register("campo").
// L'errore è collegato al campo con aria-describedby e aria-invalid.

function errorProps(id, error) {
  return error ? { "aria-invalid": true, "aria-describedby": `${id}-error` } : {};
}

function FieldError({ id, error }) {
  if (!error) {
    return null;
  }
  return (
    <div className="invalid-feedback d-block" id={`${id}-error`}>
      {error}
    </div>
  );
}

export function TextField({ id, label, registration, error, type = "text", required, hint, ...rest }) {
  return (
    <div className="form-group">
      <label htmlFor={id} className="active">
        {label}
        {required && <span aria-hidden="true"> *</span>}
      </label>
      <input
        id={id}
        type={type}
        className={`form-control ${error ? "is-invalid" : ""}`}
        required={required}
        {...errorProps(id, error)}
        {...registration}
        {...rest}
      />
      {hint && !error && <small className="form-text">{hint}</small>}
      <FieldError id={id} error={error} />
    </div>
  );
}

export function TextareaField({ id, label, registration, error, rows = 3 }) {
  return (
    <div className="form-group">
      <label htmlFor={id} className="active">
        {label}
      </label>
      <textarea
        id={id}
        rows={rows}
        className={`form-control ${error ? "is-invalid" : ""}`}
        {...errorProps(id, error)}
        {...registration}
      />
      <FieldError id={id} error={error} />
    </div>
  );
}

// options: [{ value, label }]; placeholder è la prima voce, senza valore.
export function SelectField({ id, label, registration, error, options, placeholder, required }) {
  return (
    <div className="select-wrapper">
      <label htmlFor={id}>
        {label}
        {required && <span aria-hidden="true"> *</span>}
      </label>
      <select
        id={id}
        className={error ? "is-invalid" : ""}
        required={required}
        {...errorProps(id, error)}
        {...registration}
      >
        {placeholder !== undefined && <option value="">{placeholder}</option>}
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
      <FieldError id={id} error={error} />
    </div>
  );
}
