import toast from "react-hot-toast";
import { ApiError } from "@/lib/api";

export function toastSuccess(message: string) {
  toast.success(message);
}

export function toastError(error: unknown) {
  if (error instanceof ApiError) {
    toast.error(error.message || `Error ${error.status}`);
  } else if (error instanceof Error) {
    toast.error(error.message);
  } else {
    toast.error("An unexpected error occurred");
  }
}

export function toastWarning(message: string) {
  toast(message, { icon: "⚠️" });
}

export function toastInfo(message: string) {
  toast(message, { icon: "ℹ️" });
}

export { toast };
