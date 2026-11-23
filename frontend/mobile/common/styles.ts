import { StatusBar, StyleSheet } from "react-native";

export const styles = StyleSheet.create({
  screen: { flex: 1, gap: 12, padding: 20, paddingTop: (StatusBar.currentHeight ?? 0) + 20, backgroundColor: "#f5f4f0" },
  header: { flexDirection: "row", flexWrap: "wrap", justifyContent: "space-between", alignItems: "center", gap: 8 },
  heading: { fontSize: 24, fontWeight: "600", color: "#1a1a18" },
  input: { minHeight: 48, paddingHorizontal: 12, borderColor: "#b4b2a9", borderWidth: 1, borderRadius: 8, backgroundColor: "#fff" },
  error: { color: "#a32d2d" },
});
