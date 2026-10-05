"use client";

import { BarElement, CategoryScale, Chart as ChartJS, LinearScale, Tooltip } from "chart.js";
import { Bar } from "react-chartjs-2";

// Chart.js include solo i moduli registrati: quelli di un grafico a barre.
ChartJS.register(CategoryScale, LinearScale, BarElement, Tooltip);

// Colore primario di Bootstrap Italia.
const PRIMARY = "#0066cc";

// Grafico a barre del numero di prodotti per tipo. Il canvas non è
// leggibile dai lettori di schermo: gli stessi dati sono in una tabella
// nascosta alla vista ma non a loro.
export default function ProductsByTypeChart({ data }) {
  const chartData = {
    labels: data.map((item) => item.product_type_name),
    datasets: [{ label: "Prodotti", data: data.map((item) => item.count), backgroundColor: PRIMARY }],
  };

  const options = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: { legend: { display: false } },
    scales: { y: { beginAtZero: true, ticks: { precision: 0 } } },
  };

  return (
    <>
      <div className="chart-container" aria-hidden="true">
        <Bar data={chartData} options={options} />
      </div>
      <table className="visually-hidden">
        <caption>Numero di prodotti per tipo</caption>
        <thead>
          <tr>
            <th scope="col">Tipo</th>
            <th scope="col">Prodotti</th>
          </tr>
        </thead>
        <tbody>
          {data.map((item) => (
            <tr key={item.product_type_id}>
              <th scope="row">{item.product_type_name}</th>
              <td>{item.count}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </>
  );
}
