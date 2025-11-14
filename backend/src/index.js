import express from "express";

const app = express();
const port = process.env.PORT || 3000;

app.get("/health", (req, res) => {
  res.json({ status: "ok", time: new Date().toISOString() });
});

app.listen(port, () => {
  console.log(`Test backend listening on http://0.0.0.0:${port}`);
});

