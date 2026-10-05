// Schemi di validazione dei form (zod). Riproducono le regole del backend
// (internal/user/validation.go, internal/product/validation.go, regole AGID
// in internal/user/service.go): l'utente vede subito gli errori, ma la
// verifica che conta resta quella del backend.
import { z } from "zod";

const MAX_QUANTITY = 1_000_000;

// Caratteri di controllo (a capo, tabulazioni…): non ammessi in nomi e titoli.
const CONTROL_CHARS = /\p{Cc}/u;

const requiredText = (max, label = "campo obbligatorio") =>
  z
    .string()
    .trim()
    .min(1, label)
    .max(max, `al massimo ${max} caratteri`)
    .refine((value) => !CONTROL_CHARS.test(value), "contiene caratteri non ammessi");

// Regole AGID sulle password: almeno 8 caratteri, una maiuscola, un numero e
// un carattere speciale (né lettera, né cifra, né spazio). Il limite di 72
// byte è quello di bcrypt.
export const passwordSchema = z
  .string()
  .refine(
    (value) =>
      [...value].length >= 8 &&
      new TextEncoder().encode(value).length <= 72 &&
      /\p{Lu}/u.test(value) &&
      /\p{Nd}/u.test(value) &&
      /[^\p{L}\p{Nd}\s]/u.test(value),
    "la password deve avere da 8 a 72 caratteri e contenere almeno una lettera maiuscola, un numero e un carattere speciale",
  );

// Login: solo la presenza dei campi. Una password non conforme alle regole
// AGID non può essere quella giusta: lo dirà il backend come per ogni
// password errata.
export const loginSchema = z.object({
  username: z.string().trim().min(1, "inserisci lo username"),
  password: z.string().min(1, "inserisci la password"),
});

export const forgotPasswordSchema = z.object({
  email: z.email("indirizzo email non valido").max(254, "indirizzo email non valido"),
});

export const resetPasswordSchema = z
  .object({
    password: passwordSchema,
    confirm: z.string(),
  })
  .refine((data) => data.password === data.confirm, {
    message: "le due password non coincidono",
    path: ["confirm"],
  });

// Data di nascita: dopo il 1900 e prima di oggi. Il campo è un <input
// type="date">, che restituisce AAAA-MM-GG o una stringa vuota.
const birthDate = z
  .string()
  .regex(/^\d{4}-\d{2}-\d{2}$/, "data di nascita obbligatoria")
  .refine((value) => {
    const today = new Date().toISOString().slice(0, 10);
    return value >= "1900-01-01" && value < today;
  }, "la data di nascita deve essere successiva al 1900 e precedente a oggi");

export const userSchema = z.object({
  username: z
    .string()
    .trim()
    .regex(/^[A-Za-z0-9._-]{3,50}$/, "da 3 a 50 caratteri tra lettere, numeri, punto, trattino e trattino basso"),
  email: z.email("indirizzo email non valido").max(254, "indirizzo email non valido"),
  first_name: requiredText(100),
  last_name: requiredText(100),
  date_of_birth: birthDate,
  // Usato solo dall'Admin; vuoto significa "Operatore" alla creazione.
  role_id: z.string().optional(),
});

// Numero intero da un campo del form; un campo vuoto vale 0.
const integer = (min, max, message) => z.coerce.number().int(message).min(min, message).max(max, message);

export const productSchema = z.object({
  product_type_id: z.string().min(1, "scegli il tipo di prodotto"),
  name: requiredText(200),
  description: z.string().trim().max(1000, "al massimo 1000 caratteri"),
  price: z
    .string()
    .trim()
    .regex(/^\d{1,8}(\.\d{1,2})?$/, "prezzo non valido: usa il punto per i decimali (es. 12.50), al massimo 2 decimali"),
  min_stock_threshold: integer(0, MAX_QUANTITY, "valore tra 0 e 1.000.000"),
  // Solo alla creazione: la giacenza poi cambia con carichi e scarichi.
  initial_quantity: integer(0, MAX_QUANTITY, "valore tra 0 e 1.000.000").optional(),
});

export const movementSchema = z.object({
  type: z.enum(["in", "out"], "scegli carico o scarico"),
  quantity: integer(1, MAX_QUANTITY, "valore tra 1 e 1.000.000"),
  note: z.string().trim().max(500, "al massimo 500 caratteri"),
});
