"use client";

import { useState, useCallback } from "react";
import { Plus } from "lucide-react";
import { useForm } from "react-hook-form";
import DataTable, { type Column } from "@/components/ui/DataTable";
import Button from "@/components/ui/Button";
import { Modal, ConfirmDialog } from "@/components/ui/Modal";
import { FormField, Input, Select } from "@/components/ui/FormField";
import {
  useUsers,
  useCreateUser,
  useUpdateUser,
  useDeleteUser,
  type UserFull,
  type CreateUserInput,
  type UpdateUserInput,
} from "@/lib/use-users";
import { toastSuccess, toastError } from "@/lib/toast";
import { formatRole } from "@/lib/roles";
const ROLE_OPTIONS = [
  { value: "", label: "All roles" },
  { value: "admin", label: "Admin" },
  { value: "operator", label: "Operator" },
  { value: "warehouse_manager", label: "Warehouse Manager" },
  { value: "logistics_manager", label: "Logistics Manager" },
  { value: "analyst", label: "Analyst" },
];

const ROLE_FORM_OPTIONS = ROLE_OPTIONS.filter((r) => r.value !== "");

const columns: Column<UserFull>[] = [
  {
    key: "email",
    header: "Email",
    sortable: true,
    render: (row) => (
      <span className="font-medium text-gray-900">{row.email}</span>
    ),
  },
  {
    key: "first_name",
    header: "Name",
    sortable: true,
    render: (row) => `${row.first_name} ${row.last_name}`,
  },
  {
    key: "role",
    header: "Role",
    sortable: true,
    render: (row) => (
      <span className="inline-flex items-center rounded-full bg-blue-50 px-2.5 py-0.5 text-xs font-medium text-blue-700">
        {formatRole(row.role)}
      </span>
    ),
  },
  {
    key: "created_at",
    header: "Created",
    sortable: true,
    render: (row) => new Date(row.created_at).toLocaleDateString(),
  },
];

export default function UsersPage() {
  const [page, setPage] = useState(1);
  const [sortField, setSortField] = useState("created_at");
  const [sortDesc, setSortDesc] = useState(true);
  const [roleFilter, setRoleFilter] = useState("");
  const [nameSearch, setNameSearch] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [editUser, setEditUser] = useState<UserFull | null>(null);
  const [deleteUser, setDeleteUser] = useState<UserFull | null>(null);

  const { data, isLoading } = useUsers({
    page,
    pageSize: 10,
    sortField,
    sortDesc,
    role: roleFilter || undefined,
    name: nameSearch || undefined,
  });

  const createMutation = useCreateUser();
  const updateMutation = useUpdateUser();
  const deleteMutation = useDeleteUser();

  const users = data?.data ?? [];
  const total = data?.meta?.total ?? 0;

  const handleSort = useCallback((field: string, desc: boolean) => {
    setSortField(field);
    setSortDesc(desc);
    setPage(1);
  }, []);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Users</h1>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4" />
          New User
        </Button>
      </div>

      {}
      <div className="flex flex-wrap items-center gap-3">
        <select
          value={roleFilter}
          onChange={(e) => {
            setRoleFilter(e.target.value);
            setPage(1);
          }}
          className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
        >
          {ROLE_OPTIONS.map((r) => (
            <option key={r.value} value={r.value}>
              {r.label}
            </option>
          ))}
        </select>
        <input
          type="text"
          placeholder="Search by name..."
          value={nameSearch}
          onChange={(e) => {
            setNameSearch(e.target.value);
            setPage(1);
          }}
          className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
        />
      </div>

      {}
      <DataTable<UserFull>
        columns={columns}
        data={users}
        total={total}
        page={page}
        pageSize={10}
        onPageChange={setPage}
        sortField={sortField}
        sortDesc={sortDesc}
        onSort={handleSort}
        loading={isLoading}
        emptyMessage="No users found"
        rowKey={(row) => row.id}
        onRowClick={(row) => setEditUser(row)}
      />

      {}
      <CreateUserModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onSubmit={(values) => {
          createMutation.mutate(values, {
            onSuccess: () => {
              toastSuccess("User created");
              setCreateOpen(false);
            },
            onError: toastError,
          });
        }}
        loading={createMutation.isPending}
      />

      {}
      {editUser && (
        <EditUserModal
          user={editUser}
          open={true}
          onClose={() => setEditUser(null)}
          onSubmit={(values) => {
            updateMutation.mutate(
              { id: editUser.id, ...values },
              {
                onSuccess: () => {
                  toastSuccess("User updated");
                  setEditUser(null);
                },
                onError: toastError,
              },
            );
          }}
          onDelete={() => {
            setDeleteUser(editUser);
            setEditUser(null);
          }}
          loading={updateMutation.isPending}
        />
      )}

      {}
      <ConfirmDialog
        open={!!deleteUser}
        title="Delete User"
        message={`Are you sure you want to delete ${deleteUser?.first_name} ${deleteUser?.last_name} (${deleteUser?.email})?`}
        confirmLabel="Delete"
        variant="danger"
        loading={deleteMutation.isPending}
        onConfirm={() => {
          if (!deleteUser) return;
          deleteMutation.mutate(
            { id: deleteUser.id },
            {
              onSuccess: () => {
                toastSuccess("User deleted");
                setDeleteUser(null);
              },
              onError: toastError,
            },
          );
        }}
        onClose={() => setDeleteUser(null)}
      />
    </div>
  );
}

function CreateUserModal({
  open,
  onClose,
  onSubmit,
  loading,
}: {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: CreateUserInput) => void;
  loading: boolean;
}) {
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CreateUserInput>();

  const handleClose = () => {
    reset();
    onClose();
  };

  return (
    <Modal open={open} onClose={handleClose} title="New User" size="sm">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <FormField label="Email" required error={errors.email}>
          <Input
            type="email"
            registration={register("email", { required: "Email is required" })}
            error={errors.email}
          />
        </FormField>
        <FormField label="Password" required error={errors.password}>
          <Input
            type="password"
            registration={register("password", {
              required: "Password is required",
              minLength: { value: 6, message: "Min 6 characters" },
            })}
            error={errors.password}
          />
        </FormField>
        <div className="grid grid-cols-2 gap-3">
          <FormField label="First Name" required error={errors.first_name}>
            <Input
              registration={register("first_name", {
                required: "Required",
              })}
              error={errors.first_name}
            />
          </FormField>
          <FormField label="Last Name" required error={errors.last_name}>
            <Input
              registration={register("last_name", {
                required: "Required",
              })}
              error={errors.last_name}
            />
          </FormField>
        </div>
        <FormField label="Role" required error={errors.role}>
          <Select
            registration={register("role", { required: "Role is required" })}
            options={ROLE_FORM_OPTIONS}
            placeholder="Select role"
            error={errors.role}
          />
        </FormField>
        <div className="flex justify-end gap-3 pt-2">
          <Button variant="secondary" type="button" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" loading={loading}>
            Create
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function EditUserModal({
  user,
  open,
  onClose,
  onSubmit,
  onDelete,
  loading,
}: {
  user: UserFull;
  open: boolean;
  onClose: () => void;
  onSubmit: (values: Omit<UpdateUserInput, "id">) => void;
  onDelete: () => void;
  loading: boolean;
}) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<Omit<UpdateUserInput, "id">>({
    defaultValues: {
      email: user.email,
      first_name: user.first_name,
      last_name: user.last_name,
      role: user.role,
    },
  });

  return (
    <Modal open={open} onClose={onClose} title="Edit User" size="sm">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <FormField label="Email" required error={errors.email}>
          <Input
            type="email"
            registration={register("email", { required: "Email is required" })}
            error={errors.email}
          />
        </FormField>
        <div className="grid grid-cols-2 gap-3">
          <FormField label="First Name" required error={errors.first_name}>
            <Input
              registration={register("first_name", { required: "Required" })}
              error={errors.first_name}
            />
          </FormField>
          <FormField label="Last Name" required error={errors.last_name}>
            <Input
              registration={register("last_name", { required: "Required" })}
              error={errors.last_name}
            />
          </FormField>
        </div>
        <FormField label="Role" required error={errors.role}>
          <Select
            registration={register("role", { required: "Role is required" })}
            options={ROLE_FORM_OPTIONS}
            error={errors.role}
          />
        </FormField>
        <div className="flex justify-between pt-2">
          <Button variant="danger" type="button" onClick={onDelete}>
            Delete
          </Button>
          <div className="flex gap-3">
            <Button variant="secondary" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" loading={loading}>
              Save
            </Button>
          </div>
        </div>
      </form>
    </Modal>
  );
}
