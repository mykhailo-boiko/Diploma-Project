"use client";

import { type FieldError, type UseFormRegisterReturn } from "react-hook-form";
import clsx from "clsx";

interface FormFieldProps {
  label: string;
  error?: FieldError;
  required?: boolean;
  children: React.ReactNode;
}

export function FormField({ label, error, required, children }: FormFieldProps) {
  return (
    <div>
      <label className="mb-1 block text-sm font-medium text-gray-700">
        {label}
        {required && <span className="ml-0.5 text-red-500">*</span>}
      </label>
      {children}
      {error && (
        <p className="mt-1 text-sm text-red-600">{error.message}</p>
      )}
    </div>
  );
}

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  registration?: UseFormRegisterReturn;
  error?: FieldError;
}

export function Input({ registration, error, className, ...props }: InputProps) {
  return (
    <input
      {...registration}
      {...props}
      className={clsx(
        "block w-full rounded-md border px-3 py-2 text-sm shadow-sm transition-colors",
        "focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500",
        "disabled:cursor-not-allowed disabled:bg-gray-100",
        error
          ? "border-red-300 text-red-900 placeholder:text-red-300"
          : "border-gray-300 text-gray-900 placeholder:text-gray-400",
        className,
      )}
    />
  );
}

interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  registration?: UseFormRegisterReturn;
  error?: FieldError;
  options: { value: string; label: string }[];
  placeholder?: string;
}

export function Select({
  registration,
  error,
  options,
  placeholder,
  className,
  ...props
}: SelectProps) {
  return (
    <select
      {...registration}
      {...props}
      className={clsx(
        "block w-full rounded-md border px-3 py-2 text-sm shadow-sm transition-colors",
        "focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500",
        "disabled:cursor-not-allowed disabled:bg-gray-100",
        error
          ? "border-red-300 text-red-900"
          : "border-gray-300 text-gray-900",
        className,
      )}
    >
      {placeholder && (
        <option value="">{placeholder}</option>
      )}
      {options.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  );
}

interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  registration?: UseFormRegisterReturn;
  error?: FieldError;
}

export function Textarea({ registration, error, className, ...props }: TextareaProps) {
  return (
    <textarea
      {...registration}
      {...props}
      className={clsx(
        "block w-full rounded-md border px-3 py-2 text-sm shadow-sm transition-colors",
        "focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500",
        "disabled:cursor-not-allowed disabled:bg-gray-100",
        error
          ? "border-red-300 text-red-900 placeholder:text-red-300"
          : "border-gray-300 text-gray-900 placeholder:text-gray-400",
        className,
      )}
    />
  );
}
