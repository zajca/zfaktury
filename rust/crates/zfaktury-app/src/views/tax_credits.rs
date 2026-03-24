use std::sync::Arc;

use chrono::Datelike;
use gpui::prelude::FluentBuilder;
use gpui::*;
use zfaktury_core::service::TaxCreditsService;
use zfaktury_domain::{
    Amount, DeductionCategory, DomainError, TaxChildCredit, TaxDeduction, TaxPersonalCredits,
    TaxSpouseCredit,
};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::checkbox::Checkbox;
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::components::number_input::NumberInput;
use crate::components::select::{Select, SelectOption};
use crate::components::text_input::TextInput;
use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;
use crate::util::format::format_amount;

// --- Select option builders ---

fn child_order_options() -> Vec<SelectOption> {
    vec![
        SelectOption {
            value: "1".into(),
            label: "1. dítě".into(),
        },
        SelectOption {
            value: "2".into(),
            label: "2. dítě".into(),
        },
        SelectOption {
            value: "3".into(),
            label: "3.+ dítě".into(),
        },
    ]
}

fn disability_level_options() -> Vec<SelectOption> {
    vec![
        SelectOption {
            value: "0".into(),
            label: "Žádný".into(),
        },
        SelectOption {
            value: "1".into(),
            label: "I./II. stupeň".into(),
        },
        SelectOption {
            value: "2".into(),
            label: "III. stupeň".into(),
        },
        SelectOption {
            value: "3".into(),
            label: "ZTP/P".into(),
        },
    ]
}

fn deduction_category_options() -> Vec<SelectOption> {
    vec![
        SelectOption {
            value: "mortgage".into(),
            label: "Hypotéka".into(),
        },
        SelectOption {
            value: "life_insurance".into(),
            label: "Životní pojištění".into(),
        },
        SelectOption {
            value: "pension".into(),
            label: "Penzijní připojištění".into(),
        },
        SelectOption {
            value: "donation".into(),
            label: "Dar".into(),
        },
        SelectOption {
            value: "union_dues".into(),
            label: "Odbory".into(),
        },
    ]
}

fn parse_deduction_category(s: &str) -> Option<DeductionCategory> {
    match s {
        "mortgage" => Some(DeductionCategory::Mortgage),
        "life_insurance" => Some(DeductionCategory::LifeInsurance),
        "pension" => Some(DeductionCategory::Pension),
        "donation" => Some(DeductionCategory::Donation),
        "union_dues" => Some(DeductionCategory::UnionDues),
        _ => None,
    }
}

fn deduction_category_to_value(cat: DeductionCategory) -> &'static str {
    match cat {
        DeductionCategory::Mortgage => "mortgage",
        DeductionCategory::LifeInsurance => "life_insurance",
        DeductionCategory::Pension => "pension",
        DeductionCategory::Donation => "donation",
        DeductionCategory::UnionDues => "union_dues",
    }
}

// --- Editing state structs ---

/// Inline form state for spouse credit editing.
struct SpouseEditState {
    name: Entity<TextInput>,
    birth_number: Entity<TextInput>,
    income: Entity<NumberInput>,
    ztp: Entity<Checkbox>,
    months: Entity<NumberInput>,
}

/// Inline form state for a single child credit.
struct ChildEditState {
    /// None = creating new, Some(id) = editing existing.
    editing_id: Option<i64>,
    name: Entity<TextInput>,
    birth_number: Entity<TextInput>,
    order: Entity<Select>,
    months: Entity<NumberInput>,
    ztp: Entity<Checkbox>,
}

/// Inline form state for personal credits.
struct PersonalEditState {
    is_student: Entity<Checkbox>,
    student_months: Entity<NumberInput>,
    disability_level: Entity<Select>,
}

/// Inline form state for a single deduction.
struct DeductionEditState {
    /// None = creating new, Some(id) = editing existing.
    editing_id: Option<i64>,
    category: Entity<Select>,
    description: Entity<TextInput>,
    amount: Entity<NumberInput>,
}

/// Which section is currently being edited.
enum EditingSection {
    Spouse(SpouseEditState),
    Child(ChildEditState),
    Personal(PersonalEditState),
    Deduction(DeductionEditState),
}

/// Identifies what to delete when the confirm dialog fires.
enum DeleteTarget {
    Spouse,
    Child(i64),
    Deduction(i64),
    CopyFromYear,
}

/// Helper to create a labeled form field.
fn render_labeled_field(label: &str, child: impl IntoElement) -> Div {
    div()
        .flex()
        .flex_col()
        .gap_1()
        .child(
            div()
                .text_xs()
                .font_weight(FontWeight::MEDIUM)
                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                .child(label.to_string()),
        )
        .child(child)
}

/// Tax credits and deductions view.
pub struct TaxCreditsView {
    service: Arc<TaxCreditsService>,
    year: i32,
    loading: bool,
    error: Option<String>,
    spouse_credit: Option<TaxSpouseCredit>,
    children: Vec<TaxChildCredit>,
    personal: Option<TaxPersonalCredits>,
    deductions: Vec<TaxDeduction>,
    saving: bool,
    editing: Option<EditingSection>,
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    pending_delete: Option<DeleteTarget>,
}

impl TaxCreditsView {
    pub fn new(service: Arc<TaxCreditsService>, cx: &mut Context<Self>) -> Self {
        let year = chrono::Local::now().date_naive().year();
        let mut view = Self {
            service,
            year,
            loading: true,
            error: None,
            spouse_credit: None,
            children: Vec::new(),
            personal: None,
            deductions: Vec::new(),
            saving: false,
            editing: None,
            confirm_dialog: None,
            pending_delete: None,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        self.loading = true;
        cx.notify();
        let service = self.service.clone();
        let year = self.year;

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    let spouse = service.get_spouse(year).ok();
                    let children = service.list_children(year)?;
                    let personal = service.get_personal(year).ok();
                    let deductions = service.list_deductions(year)?;
                    Ok::<
                        (
                            Option<TaxSpouseCredit>,
                            Vec<TaxChildCredit>,
                            Option<TaxPersonalCredits>,
                            Vec<TaxDeduction>,
                        ),
                        DomainError,
                    >((spouse, children, personal, deductions))
                })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok((spouse, children, personal, deductions)) => {
                        this.spouse_credit = spouse;
                        this.children = children;
                        this.personal = personal;
                        this.deductions = deductions;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba při načítání slev: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn change_year(&mut self, delta: i32, cx: &mut Context<Self>) {
        self.year += delta;
        self.loading = true;
        self.error = None;
        self.editing = None;
        self.pending_delete = None;
        self.confirm_dialog = None;
        cx.notify();
        self.load_data(cx);
    }

    // --- Spouse CRUD ---

    fn start_add_spouse(&mut self, cx: &mut Context<Self>) {
        let name = cx.new(|cx| TextInput::new("spouse-name", "Jméno manželky/manžela...", cx));
        let birth_number = cx.new(|cx| TextInput::new("spouse-bn", "XXXXXX/XXXX", cx));
        let income = cx.new(|cx| NumberInput::new("spouse-income", "0.00", cx));
        let ztp = cx.new(|_cx| Checkbox::new("spouse-ztp", "Držitel ZTP/P", false));
        let months = cx.new(|cx| {
            NumberInput::new("spouse-months", "12", cx)
                .integer_only()
                .with_value("12")
        });
        self.editing = Some(EditingSection::Spouse(SpouseEditState {
            name,
            birth_number,
            income,
            ztp,
            months,
        }));
        self.error = None;
        cx.notify();
    }

    fn start_edit_spouse(&mut self, cx: &mut Context<Self>) {
        let sc = match &self.spouse_credit {
            Some(sc) => sc.clone(),
            None => return,
        };
        let name = cx.new(|cx| {
            let mut t = TextInput::new("spouse-name-edit", "Jméno manželky/manžela...", cx);
            t.set_value(&sc.spouse_name, cx);
            t
        });
        let birth_number = cx.new(|cx| {
            let mut t = TextInput::new("spouse-bn-edit", "XXXXXX/XXXX", cx);
            t.set_value(&sc.spouse_birth_number, cx);
            t
        });
        let income = cx.new(|cx| {
            let mut n = NumberInput::new("spouse-income-edit", "0.00", cx);
            n.set_amount(sc.spouse_income, cx);
            n
        });
        let ztp = cx.new(|_cx| Checkbox::new("spouse-ztp-edit", "Držitel ZTP/P", sc.spouse_ztp));
        let months = cx.new(|cx| {
            NumberInput::new("spouse-months-edit", "12", cx)
                .integer_only()
                .with_value(sc.months_claimed.to_string())
        });
        self.editing = Some(EditingSection::Spouse(SpouseEditState {
            name,
            birth_number,
            income,
            ztp,
            months,
        }));
        self.error = None;
        cx.notify();
    }

    fn save_spouse(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }
        let state = match &self.editing {
            Some(EditingSection::Spouse(s)) => s,
            _ => return,
        };

        let name = state.name.read(cx).value().to_string();
        let birth_number = state.birth_number.read(cx).value().to_string();
        let income = state.income.read(cx).to_amount().unwrap_or(Amount::ZERO);
        let ztp = state.ztp.read(cx).is_checked();
        let months: i32 = state.months.read(cx).value().parse().unwrap_or(12);

        if name.trim().is_empty() || birth_number.trim().is_empty() {
            self.error = Some("Jméno a rodné číslo jsou povinné.".into());
            cx.notify();
            return;
        }
        if !(1..=12).contains(&months) {
            self.error = Some("Počet měsíců musí být 1-12.".into());
            cx.notify();
            return;
        }

        let now = chrono::Local::now().naive_local();
        let mut credit = TaxSpouseCredit {
            id: self.spouse_credit.as_ref().map(|s| s.id).unwrap_or(0),
            year: self.year,
            spouse_name: name,
            spouse_birth_number: birth_number,
            spouse_income: income,
            spouse_ztp: ztp,
            months_claimed: months,
            credit_amount: Amount::ZERO,
            created_at: now,
            updated_at: now,
        };

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    service.upsert_spouse(&mut credit)?;
                    Ok::<(), DomainError>(())
                })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(()) => {
                        this.editing = None;
                        this.load_data(cx);
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba při ukládání: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn request_delete_spouse(&mut self, cx: &mut Context<Self>) {
        self.pending_delete = Some(DeleteTarget::Spouse);
        let dialog = cx.new(|_cx| {
            ConfirmDialog::new(
                "Smazat slevu na manžela/ku",
                "Opravdu chcete smazat slevu na manžela/ku?",
                "Smazat",
            )
        });
        cx.subscribe(&dialog, Self::on_confirm_result).detach();
        self.confirm_dialog = Some(dialog);
        cx.notify();
    }

    fn do_delete_spouse(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let year = self.year;
        self.saving = true;
        cx.notify();

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.delete_spouse(year) })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(()) => {
                        this.editing = None;
                        this.load_data(cx);
                    }
                    Err(e) => this.error = Some(format!("Chyba při mazání: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    // --- Child CRUD ---

    fn start_add_child(&mut self, cx: &mut Context<Self>) {
        let name = cx.new(|cx| TextInput::new("child-name-new", "Jméno dítěte...", cx));
        let birth_number = cx.new(|cx| TextInput::new("child-bn-new", "XXXXXX/XXXX", cx));
        let order =
            cx.new(|_cx| Select::new("child-order-new", "Pořadí...", child_order_options()));
        let months = cx.new(|cx| {
            NumberInput::new("child-months-new", "12", cx)
                .integer_only()
                .with_value("12")
        });
        let ztp = cx.new(|_cx| Checkbox::new("child-ztp-new", "Držitel ZTP/P", false));

        self.editing = Some(EditingSection::Child(ChildEditState {
            editing_id: None,
            name,
            birth_number,
            order,
            months,
            ztp,
        }));
        self.error = None;
        cx.notify();
    }

    fn start_edit_child(&mut self, child: &TaxChildCredit, cx: &mut Context<Self>) {
        let child_id = child.id;
        let name = cx.new(|cx| {
            let mut t = TextInput::new(
                SharedString::from(format!("child-name-{}", child_id)),
                "Jméno dítěte...",
                cx,
            );
            t.set_value(&child.child_name, cx);
            t
        });
        let birth_number = cx.new(|cx| {
            let mut t = TextInput::new(
                SharedString::from(format!("child-bn-{}", child_id)),
                "XXXXXX/XXXX",
                cx,
            );
            t.set_value(&child.birth_number, cx);
            t
        });
        let order = cx.new(|cx| {
            let mut sel = Select::new(
                SharedString::from(format!("child-order-{}", child_id)),
                "Pořadí...",
                child_order_options(),
            );
            sel.set_selected_value(&child.child_order.to_string(), cx);
            sel
        });
        let months = cx.new(|cx| {
            NumberInput::new(
                SharedString::from(format!("child-months-{}", child_id)),
                "12",
                cx,
            )
            .integer_only()
            .with_value(child.months_claimed.to_string())
        });
        let ztp = cx.new(|_cx| {
            Checkbox::new(
                SharedString::from(format!("child-ztp-{}", child_id)),
                "Držitel ZTP/P",
                child.ztp,
            )
        });

        self.editing = Some(EditingSection::Child(ChildEditState {
            editing_id: Some(child_id),
            name,
            birth_number,
            order,
            months,
            ztp,
        }));
        self.error = None;
        cx.notify();
    }

    fn save_child(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }
        let state = match &self.editing {
            Some(EditingSection::Child(s)) => s,
            _ => return,
        };

        let name = state.name.read(cx).value().to_string();
        let birth_number = state.birth_number.read(cx).value().to_string();
        let order: i32 = state
            .order
            .read(cx)
            .selected_value()
            .and_then(|v| v.parse().ok())
            .unwrap_or(0);
        let months: i32 = state.months.read(cx).value().parse().unwrap_or(12);
        let ztp = state.ztp.read(cx).is_checked();
        let editing_id = state.editing_id;

        if name.trim().is_empty() || birth_number.trim().is_empty() {
            self.error = Some("Jméno a rodné číslo jsou povinné.".into());
            cx.notify();
            return;
        }
        if !(1..=3).contains(&order) {
            self.error = Some("Vyberte pořadí dítěte.".into());
            cx.notify();
            return;
        }
        if !(1..=12).contains(&months) {
            self.error = Some("Počet měsíců musí být 1-12.".into());
            cx.notify();
            return;
        }

        let now = chrono::Local::now().naive_local();
        let mut credit = TaxChildCredit {
            id: editing_id.unwrap_or(0),
            year: self.year,
            child_name: name,
            birth_number,
            child_order: order,
            months_claimed: months,
            ztp,
            credit_amount: Amount::ZERO,
            created_at: now,
            updated_at: now,
        };

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        let is_new = editing_id.is_none();

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    if is_new {
                        service.create_child(&mut credit)?;
                    } else {
                        service.update_child(&mut credit)?;
                    }
                    Ok::<(), DomainError>(())
                })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(()) => {
                        this.editing = None;
                        this.load_data(cx);
                    }
                    Err(e) => this.error = Some(format!("Chyba při ukládání: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn request_delete_child(&mut self, id: i64, cx: &mut Context<Self>) {
        self.pending_delete = Some(DeleteTarget::Child(id));
        let dialog = cx.new(|_cx| {
            ConfirmDialog::new("Smazat dítě", "Opravdu chcete smazat toto dítě?", "Smazat")
        });
        cx.subscribe(&dialog, Self::on_confirm_result).detach();
        self.confirm_dialog = Some(dialog);
        cx.notify();
    }

    fn do_delete_child(&mut self, id: i64, cx: &mut Context<Self>) {
        let service = self.service.clone();
        self.saving = true;
        cx.notify();

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.delete_child(id) })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(()) => {
                        if let Some(EditingSection::Child(ref state)) = this.editing
                            && state.editing_id == Some(id)
                        {
                            this.editing = None;
                        }
                        this.load_data(cx);
                    }
                    Err(e) => this.error = Some(format!("Chyba při mazání: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    // --- Personal credits CRUD ---

    fn start_edit_personal(&mut self, cx: &mut Context<Self>) {
        let (is_student_val, student_months_val, disability_val) = match &self.personal {
            Some(pc) => (pc.is_student, pc.student_months, pc.disability_level),
            None => (false, 0, 0),
        };

        let is_student = cx.new(|_cx| Checkbox::new("personal-student", "Student", is_student_val));
        let student_months = cx.new(|cx| {
            NumberInput::new("personal-student-months", "12", cx)
                .integer_only()
                .with_value(student_months_val.to_string())
        });
        let disability_level = cx.new(|cx| {
            let mut sel = Select::new(
                "personal-disability",
                "Stupeň invalidity...",
                disability_level_options(),
            );
            sel.set_selected_value(&disability_val.to_string(), cx);
            sel
        });

        self.editing = Some(EditingSection::Personal(PersonalEditState {
            is_student,
            student_months,
            disability_level,
        }));
        self.error = None;
        cx.notify();
    }

    fn save_personal(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }
        let state = match &self.editing {
            Some(EditingSection::Personal(s)) => s,
            _ => return,
        };

        let is_student = state.is_student.read(cx).is_checked();
        let student_months: i32 = if is_student {
            state.student_months.read(cx).value().parse().unwrap_or(0)
        } else {
            0
        };
        let disability_level: i32 = state
            .disability_level
            .read(cx)
            .selected_value()
            .and_then(|v| v.parse().ok())
            .unwrap_or(0);

        if is_student && !(0..=12).contains(&student_months) {
            self.error = Some("Počet měsíců studia musí být 0-12.".into());
            cx.notify();
            return;
        }

        let now = chrono::Local::now().naive_local();
        let mut credits = TaxPersonalCredits {
            year: self.year,
            is_student,
            student_months,
            disability_level,
            credit_student: Amount::ZERO,
            credit_disability: Amount::ZERO,
            created_at: now,
            updated_at: now,
        };

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.upsert_personal(&mut credits) })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(()) => {
                        this.editing = None;
                        this.load_data(cx);
                    }
                    Err(e) => this.error = Some(format!("Chyba při ukládání: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    // --- Deduction CRUD ---

    fn start_add_deduction(&mut self, cx: &mut Context<Self>) {
        let category = cx.new(|_cx| {
            Select::new(
                "ded-category-new",
                "Kategorie...",
                deduction_category_options(),
            )
        });
        let description = cx.new(|cx| TextInput::new("ded-desc-new", "Popis odpočtu...", cx));
        let amount = cx.new(|cx| NumberInput::new("ded-amount-new", "0.00", cx));

        self.editing = Some(EditingSection::Deduction(DeductionEditState {
            editing_id: None,
            category,
            description,
            amount,
        }));
        self.error = None;
        cx.notify();
    }

    fn start_edit_deduction(&mut self, ded: &TaxDeduction, cx: &mut Context<Self>) {
        let ded_id = ded.id;
        let category = cx.new(|cx| {
            let mut sel = Select::new(
                SharedString::from(format!("ded-category-{}", ded_id)),
                "Kategorie...",
                deduction_category_options(),
            );
            sel.set_selected_value(deduction_category_to_value(ded.category), cx);
            sel
        });
        let description = cx.new(|cx| {
            let mut t = TextInput::new(
                SharedString::from(format!("ded-desc-{}", ded_id)),
                "Popis odpočtu...",
                cx,
            );
            t.set_value(&ded.description, cx);
            t
        });
        let amount = cx.new(|cx| {
            let mut n = NumberInput::new(
                SharedString::from(format!("ded-amount-{}", ded_id)),
                "0.00",
                cx,
            );
            n.set_amount(ded.claimed_amount, cx);
            n
        });

        self.editing = Some(EditingSection::Deduction(DeductionEditState {
            editing_id: Some(ded_id),
            category,
            description,
            amount,
        }));
        self.error = None;
        cx.notify();
    }

    fn save_deduction(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }
        let state = match &self.editing {
            Some(EditingSection::Deduction(s)) => s,
            _ => return,
        };

        let category_str = state
            .category
            .read(cx)
            .selected_value()
            .unwrap_or("")
            .to_string();
        let description = state.description.read(cx).value().to_string();
        let claimed_amount = state.amount.read(cx).to_amount().unwrap_or(Amount::ZERO);
        let editing_id = state.editing_id;

        let category = match parse_deduction_category(&category_str) {
            Some(c) => c,
            None => {
                self.error = Some("Vyberte kategorii odpočtu.".into());
                cx.notify();
                return;
            }
        };

        if description.trim().is_empty() {
            self.error = Some("Popis odpočtu je povinný.".into());
            cx.notify();
            return;
        }
        if claimed_amount.halere() <= 0 {
            self.error = Some("Částka musí být větší než 0.".into());
            cx.notify();
            return;
        }

        let now = chrono::Local::now().naive_local();
        let mut ded = TaxDeduction {
            id: editing_id.unwrap_or(0),
            year: self.year,
            category,
            description,
            claimed_amount,
            max_amount: Amount::ZERO,
            allowed_amount: Amount::ZERO,
            created_at: now,
            updated_at: now,
        };

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        let is_new = editing_id.is_none();

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    if is_new {
                        service.create_deduction(&mut ded)?;
                    } else {
                        service.update_deduction(&mut ded)?;
                    }
                    Ok::<(), DomainError>(())
                })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(()) => {
                        this.editing = None;
                        this.load_data(cx);
                    }
                    Err(e) => this.error = Some(format!("Chyba při ukládání: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn request_delete_deduction(&mut self, id: i64, cx: &mut Context<Self>) {
        self.pending_delete = Some(DeleteTarget::Deduction(id));
        let dialog = cx.new(|_cx| {
            ConfirmDialog::new(
                "Smazat odpočet",
                "Opravdu chcete smazat tento odpočet?",
                "Smazat",
            )
        });
        cx.subscribe(&dialog, Self::on_confirm_result).detach();
        self.confirm_dialog = Some(dialog);
        cx.notify();
    }

    fn do_delete_deduction(&mut self, id: i64, cx: &mut Context<Self>) {
        let service = self.service.clone();
        self.saving = true;
        cx.notify();

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.delete_deduction(id) })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(()) => {
                        if let Some(EditingSection::Deduction(ref state)) = this.editing
                            && state.editing_id == Some(id)
                        {
                            this.editing = None;
                        }
                        this.load_data(cx);
                    }
                    Err(e) => this.error = Some(format!("Chyba při mazání: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    // --- Copy from previous year ---

    fn request_copy_from_previous_year(&mut self, cx: &mut Context<Self>) {
        let source_year = self.year - 1;
        self.pending_delete = Some(DeleteTarget::CopyFromYear);
        let dialog = cx.new(|_cx| {
            ConfirmDialog::new(
                format!("Kopírovat z roku {}", source_year),
                format!(
                    "Kopírovat slevy a odpočty z roku {}? Existující data pro rok {} budou nahrazena.",
                    source_year, source_year + 1
                ),
                "Kopírovat",
            )
        });
        cx.subscribe(&dialog, Self::on_confirm_result).detach();
        self.confirm_dialog = Some(dialog);
        cx.notify();
    }

    fn copy_from_previous_year(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let target_year = self.year;
        let source_year = target_year - 1;

        self.saving = true;
        self.error = None;
        cx.notify();

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    let spouse = service.get_spouse(source_year).ok();
                    let children = service.list_children(source_year)?;
                    let personal = service.get_personal(source_year).ok();
                    let deductions = service.list_deductions(source_year)?;

                    if let Some(sc) = spouse {
                        let mut new_sc = sc.clone();
                        new_sc.id = 0;
                        new_sc.year = target_year;
                        service.upsert_spouse(&mut new_sc)?;
                    }

                    for child in &children {
                        let mut new_child = child.clone();
                        new_child.id = 0;
                        new_child.year = target_year;
                        service.create_child(&mut new_child)?;
                    }

                    if let Some(pc) = personal {
                        let mut new_pc = pc.clone();
                        new_pc.year = target_year;
                        service.upsert_personal(&mut new_pc)?;
                    }

                    for ded in &deductions {
                        let mut new_ded = ded.clone();
                        new_ded.id = 0;
                        new_ded.year = target_year;
                        service.create_deduction(&mut new_ded)?;
                    }

                    Ok::<(), DomainError>(())
                })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(()) => this.load_data(cx),
                    Err(e) => {
                        this.error = Some(format!("Chyba při kopírování: {e}"));
                        cx.notify();
                    }
                }
            })
            .ok();
        })
        .detach();
    }

    // --- Confirm dialog handler ---

    fn on_confirm_result(
        &mut self,
        _dialog: Entity<ConfirmDialog>,
        event: &ConfirmDialogResult,
        cx: &mut Context<Self>,
    ) {
        match event {
            ConfirmDialogResult::Confirmed => {
                if let Some(target) = self.pending_delete.take() {
                    self.confirm_dialog = None;
                    match target {
                        DeleteTarget::Spouse => self.do_delete_spouse(cx),
                        DeleteTarget::Child(id) => self.do_delete_child(id, cx),
                        DeleteTarget::Deduction(id) => self.do_delete_deduction(id, cx),
                        DeleteTarget::CopyFromYear => self.copy_from_previous_year(cx),
                    }
                }
            }
            ConfirmDialogResult::Cancelled => {
                self.pending_delete = None;
                self.confirm_dialog = None;
                cx.notify();
            }
        }
    }

    fn cancel_edit(&mut self, cx: &mut Context<Self>) {
        self.editing = None;
        self.error = None;
        cx.notify();
    }

    // --- Render helpers ---

    fn render_spouse_card(&mut self, cx: &mut Context<Self>) -> Div {
        let has_editing = self.editing.is_some();
        let is_editing_spouse = matches!(&self.editing, Some(EditingSection::Spouse(_)));

        let mut card = div()
            .p_4()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .flex_col()
            .gap_2();

        // Card header with title and action buttons
        let mut header = div().flex().items_center().justify_between().child(
            div()
                .flex()
                .flex_col()
                .gap_1()
                .child(
                    div()
                        .text_sm()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Sleva na manželku/manžela"),
                )
                .child(
                    div()
                        .text_xs()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("Příjem manželky/manžela do 68 000 Kč. ZTP/P = dvojnásobek."),
                ),
        );

        if !is_editing_spouse {
            let mut buttons = div().flex().gap_1();
            if self.spouse_credit.is_some() {
                buttons = buttons
                    .child(render_button(
                        "spouse-edit-btn",
                        "Upravit",
                        ButtonVariant::Secondary,
                        has_editing || self.saving,
                        false,
                        cx.listener(|this, _event: &ClickEvent, _window, cx| {
                            this.start_edit_spouse(cx);
                        }),
                    ))
                    .child(render_button(
                        "spouse-delete-btn",
                        "Smazat",
                        ButtonVariant::Danger,
                        has_editing || self.saving,
                        false,
                        cx.listener(|this, _event: &ClickEvent, _window, cx| {
                            this.request_delete_spouse(cx);
                        }),
                    ));
            } else {
                buttons = buttons.child(render_button(
                    "spouse-add-btn",
                    "Přidat",
                    ButtonVariant::Primary,
                    has_editing || self.saving,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.start_add_spouse(cx);
                    }),
                ));
            }
            header = header.child(buttons);
        }

        card = card.child(header);

        // Form or display mode
        if is_editing_spouse {
            if let Some(EditingSection::Spouse(ref state)) = self.editing {
                card = card
                    .child(render_labeled_field("Jméno", state.name.clone()))
                    .child(render_labeled_field(
                        "Rodné číslo",
                        state.birth_number.clone(),
                    ))
                    .child(render_labeled_field("Příjem", state.income.clone()))
                    .child(state.ztp.clone())
                    .child(render_labeled_field("Měsíců", state.months.clone()))
                    .child(
                        div()
                            .flex()
                            .justify_end()
                            .gap_2()
                            .pt_2()
                            .child(render_button(
                                "spouse-cancel",
                                "Zrušit",
                                ButtonVariant::Secondary,
                                self.saving,
                                false,
                                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                    this.cancel_edit(cx);
                                }),
                            ))
                            .child(render_button(
                                "spouse-save",
                                "Uložit",
                                ButtonVariant::Primary,
                                false,
                                self.saving,
                                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                    this.save_spouse(cx);
                                }),
                            )),
                    );
            }
        } else if let Some(ref sc) = self.spouse_credit {
            card = card
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child("Jméno"),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(sc.spouse_name.clone()),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child("Příjem"),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(sc.spouse_income)),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child("ZTP/P"),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(if sc.spouse_ztp { "Ano" } else { "Ne" }),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child("Měsíců"),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(sc.months_claimed.to_string()),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child("Sleva"),
                        )
                        .child(
                            div()
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(rgb(ZfColors::ACCENT))
                                .child(format_amount(sc.credit_amount)),
                        ),
                );
        } else {
            card = card.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Žádná sleva na manžela/ku pro tento rok."),
            );
        }

        card
    }

    fn render_children_card(&mut self, cx: &mut Context<Self>) -> Div {
        let has_editing = self.editing.is_some();
        let is_editing_child = matches!(&self.editing, Some(EditingSection::Child(_)));

        let mut card = div()
            .p_4()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .flex_col()
            .gap_2();

        // Card header
        let mut header = div().flex().items_center().justify_between().child(
            div()
                .flex()
                .flex_col()
                .gap_1()
                .child(
                    div()
                        .text_sm()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Děti"),
                )
                .child(
                    div()
                        .text_xs()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("Daňové zvýhodnění na vyživované děti. 1./2./3.+ dítě."),
                ),
        );

        if !is_editing_child {
            header = header.child(render_button(
                "child-add-btn",
                "Přidat dítě",
                ButtonVariant::Primary,
                has_editing || self.saving,
                false,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.start_add_child(cx);
                }),
            ));
        }

        card = card.child(header);

        // Children list
        if self.children.is_empty() && !is_editing_child {
            card = card.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Žádné děti pro tento rok."),
            );
        } else {
            // Clone children data to avoid borrow issues
            let children_data: Vec<(i64, i32, String, i32, bool, Amount)> = self
                .children
                .iter()
                .map(|c| {
                    (
                        c.id,
                        c.child_order,
                        c.child_name.clone(),
                        c.months_claimed,
                        c.ztp,
                        c.credit_amount,
                    )
                })
                .collect();

            for (child_id, child_order, child_name, months_claimed, ztp, credit_amount) in
                children_data
            {
                let row = div()
                    .flex()
                    .items_center()
                    .justify_between()
                    .text_sm()
                    .py_1()
                    .border_t_1()
                    .border_color(rgb(ZfColors::BORDER_SUBTLE))
                    .child(div().text_color(rgb(ZfColors::TEXT_PRIMARY)).child(format!(
                        "{}. dítě: {} ({}m{})",
                        child_order,
                        child_name,
                        months_claimed,
                        if ztp { " ZTP/P" } else { "" }
                    )))
                    .child(
                        div()
                            .flex()
                            .items_center()
                            .gap_2()
                            .child(
                                div()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::ACCENT))
                                    .child(format_amount(credit_amount)),
                            )
                            .child(render_button(
                                SharedString::from(format!("child-edit-{}", child_id)),
                                "Upravit",
                                ButtonVariant::Secondary,
                                has_editing || self.saving,
                                false,
                                cx.listener(move |this, _event: &ClickEvent, _window, cx| {
                                    if let Some(child) =
                                        this.children.iter().find(|c| c.id == child_id)
                                    {
                                        let child = child.clone();
                                        this.start_edit_child(&child, cx);
                                    }
                                }),
                            ))
                            .child(render_button(
                                SharedString::from(format!("child-del-{}", child_id)),
                                "Smazat",
                                ButtonVariant::Danger,
                                has_editing || self.saving,
                                false,
                                cx.listener(move |this, _event: &ClickEvent, _window, cx| {
                                    this.request_delete_child(child_id, cx);
                                }),
                            )),
                    );

                card = card.child(row);
            }
        }

        // Inline form for adding/editing child
        if is_editing_child && let Some(EditingSection::Child(ref state)) = self.editing {
            card = card.child(
                div()
                    .border_t_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .pt_3()
                    .mt_1()
                    .flex()
                    .flex_col()
                    .gap_2()
                    .child(
                        div()
                            .text_xs()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child(if state.editing_id.is_some() {
                                "Upravit dítě"
                            } else {
                                "Přidat dítě"
                            }),
                    )
                    .child(render_labeled_field("Jméno", state.name.clone()))
                    .child(render_labeled_field(
                        "Rodné číslo",
                        state.birth_number.clone(),
                    ))
                    .child(render_labeled_field("Pořadí", state.order.clone()))
                    .child(render_labeled_field("Měsíců", state.months.clone()))
                    .child(state.ztp.clone())
                    .child(
                        div()
                            .flex()
                            .justify_end()
                            .gap_2()
                            .pt_2()
                            .child(render_button(
                                "child-cancel",
                                "Zrušit",
                                ButtonVariant::Secondary,
                                self.saving,
                                false,
                                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                    this.cancel_edit(cx);
                                }),
                            ))
                            .child(render_button(
                                "child-save",
                                "Uložit",
                                ButtonVariant::Primary,
                                false,
                                self.saving,
                                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                    this.save_child(cx);
                                }),
                            )),
                    ),
            );
        }

        card
    }

    fn render_personal_card(&mut self, cx: &mut Context<Self>) -> Div {
        let has_editing = self.editing.is_some();
        let is_editing_personal = matches!(&self.editing, Some(EditingSection::Personal(_)));

        let mut card = div()
            .p_4()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .flex_col()
            .gap_2();

        // Card header
        let mut header = div().flex().items_center().justify_between().child(
            div()
                .flex()
                .flex_col()
                .gap_1()
                .child(
                    div()
                        .text_sm()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Osobní slevy"),
                )
                .child(
                    div()
                        .text_xs()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("Sleva na studenta, invalidní důchod."),
                ),
        );

        if !is_editing_personal {
            header = header.child(render_button(
                "personal-edit-btn",
                "Upravit",
                ButtonVariant::Secondary,
                has_editing || self.saving,
                false,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.start_edit_personal(cx);
                }),
            ));
        }

        card = card.child(header);

        // Form or display mode
        if is_editing_personal {
            if let Some(EditingSection::Personal(ref state)) = self.editing {
                let is_student = state.is_student.read(cx).is_checked();

                card =
                    card.child(state.is_student.clone())
                        .child(div().when(!is_student, |d| d.opacity(0.5)).child(
                            render_labeled_field("Měsíců studia", state.student_months.clone()),
                        ))
                        .child(render_labeled_field(
                            "Stupeň invalidity",
                            state.disability_level.clone(),
                        ))
                        .child(
                            div()
                                .flex()
                                .justify_end()
                                .gap_2()
                                .pt_2()
                                .child(render_button(
                                    "personal-cancel",
                                    "Zrušit",
                                    ButtonVariant::Secondary,
                                    self.saving,
                                    false,
                                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                        this.cancel_edit(cx);
                                    }),
                                ))
                                .child(render_button(
                                    "personal-save",
                                    "Uložit",
                                    ButtonVariant::Primary,
                                    false,
                                    self.saving,
                                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                        this.save_personal(cx);
                                    }),
                                )),
                        );
            }
        } else if let Some(ref pc) = self.personal {
            if pc.is_student {
                card = card.child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format!("Student ({}m)", pc.student_months)),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(pc.credit_student)),
                        ),
                );
            }
            if pc.disability_level > 0 {
                let level_text = match pc.disability_level {
                    1 => "I./II. stupeň",
                    2 => "III. stupeň",
                    3 => "ZTP/P",
                    _ => "Neznámé",
                };
                card = card.child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format!("Invalidita ({})", level_text)),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(pc.credit_disability)),
                        ),
                );
            }
            if !pc.is_student && pc.disability_level == 0 {
                card = card.child(
                    div()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("Žádné osobní slevy."),
                );
            }
        } else {
            card = card.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Žádné osobní slevy pro tento rok."),
            );
        }

        card
    }

    fn render_deductions_card(&mut self, cx: &mut Context<Self>) -> Div {
        let has_editing = self.editing.is_some();
        let is_editing_deduction = matches!(&self.editing, Some(EditingSection::Deduction(_)));

        let mut card = div()
            .p_4()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .flex_col()
            .gap_2();

        // Card header
        let mut header =
            div().flex().items_center().justify_between().child(
                div()
                    .flex()
                    .flex_col()
                    .gap_1()
                    .child(
                        div()
                            .text_sm()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child("Odpočty"),
                    )
                    .child(div().text_xs().text_color(rgb(ZfColors::TEXT_MUTED)).child(
                        "Hypotéka, životní pojištění, penzijní připojištění, dary, odbory.",
                    )),
            );

        if !is_editing_deduction {
            header = header.child(render_button(
                "ded-add-btn",
                "Přidat odpočet",
                ButtonVariant::Primary,
                has_editing || self.saving,
                false,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.start_add_deduction(cx);
                }),
            ));
        }

        card = card.child(header);

        // Deductions list
        if self.deductions.is_empty() && !is_editing_deduction {
            card = card.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Žádné odpočty pro tento rok."),
            );
        } else {
            // Clone deduction data to avoid borrow issues
            let deductions_data: Vec<(i64, DeductionCategory, String, Amount)> = self
                .deductions
                .iter()
                .map(|d| (d.id, d.category, d.description.clone(), d.allowed_amount))
                .collect();

            for (ded_id, category, description, allowed_amount) in deductions_data {
                let category_label = match category {
                    DeductionCategory::Mortgage => "Hypotéka",
                    DeductionCategory::LifeInsurance => "Životní pojištění",
                    DeductionCategory::Pension => "Penzijní připojištění",
                    DeductionCategory::Donation => "Dar",
                    DeductionCategory::UnionDues => "Odbory",
                };
                card = card.child(
                    div()
                        .flex()
                        .items_center()
                        .justify_between()
                        .text_sm()
                        .py_1()
                        .border_t_1()
                        .border_color(rgb(ZfColors::BORDER_SUBTLE))
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format!("{}: {}", category_label, description)),
                        )
                        .child(
                            div()
                                .flex()
                                .items_center()
                                .gap_2()
                                .child(
                                    div()
                                        .font_weight(FontWeight::MEDIUM)
                                        .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                        .child(format_amount(allowed_amount)),
                                )
                                .child(render_button(
                                    SharedString::from(format!("ded-edit-{}", ded_id)),
                                    "Upravit",
                                    ButtonVariant::Secondary,
                                    has_editing || self.saving,
                                    false,
                                    cx.listener(move |this, _event: &ClickEvent, _window, cx| {
                                        if let Some(ded) =
                                            this.deductions.iter().find(|d| d.id == ded_id)
                                        {
                                            let ded = ded.clone();
                                            this.start_edit_deduction(&ded, cx);
                                        }
                                    }),
                                ))
                                .child(render_button(
                                    SharedString::from(format!("ded-del-{}", ded_id)),
                                    "Smazat",
                                    ButtonVariant::Danger,
                                    has_editing || self.saving,
                                    false,
                                    cx.listener(move |this, _event: &ClickEvent, _window, cx| {
                                        this.request_delete_deduction(ded_id, cx);
                                    }),
                                )),
                        ),
                );
            }
        }

        // Inline form for adding/editing deduction
        if is_editing_deduction && let Some(EditingSection::Deduction(ref state)) = self.editing {
            card = card.child(
                div()
                    .border_t_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .pt_3()
                    .mt_1()
                    .flex()
                    .flex_col()
                    .gap_2()
                    .child(
                        div()
                            .text_xs()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child(if state.editing_id.is_some() {
                                "Upravit odpočet"
                            } else {
                                "Přidat odpočet"
                            }),
                    )
                    .child(render_labeled_field("Kategorie", state.category.clone()))
                    .child(render_labeled_field("Popis", state.description.clone()))
                    .child(render_labeled_field("Částka", state.amount.clone()))
                    .child(
                        div()
                            .flex()
                            .justify_end()
                            .gap_2()
                            .pt_2()
                            .child(render_button(
                                "ded-cancel",
                                "Zrušit",
                                ButtonVariant::Secondary,
                                self.saving,
                                false,
                                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                    this.cancel_edit(cx);
                                }),
                            ))
                            .child(render_button(
                                "ded-save",
                                "Uložit",
                                ButtonVariant::Primary,
                                false,
                                self.saving,
                                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                    this.save_deduction(cx);
                                }),
                            )),
                    ),
            );
        }

        card
    }
}

impl EventEmitter<NavigateEvent> for TaxCreditsView {}

impl Render for TaxCreditsView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let has_editing = self.editing.is_some();

        let mut outer = div()
            .id("tax-credits-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll();

        // Header with year selector and copy button
        outer = outer.child(
            div()
                .flex()
                .items_center()
                .gap_3()
                .child(
                    div()
                        .text_xl()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Slevy a odpočty"),
                )
                .child(render_button(
                    "btn-year-prev",
                    "<",
                    ButtonVariant::Secondary,
                    self.saving,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.change_year(-1, cx);
                    }),
                ))
                .child(
                    div()
                        .px_3()
                        .py_1()
                        .bg(rgb(ZfColors::SURFACE))
                        .border_1()
                        .border_color(rgb(ZfColors::BORDER))
                        .rounded_md()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child(self.year.to_string()),
                )
                .child(render_button(
                    "btn-year-next",
                    ">",
                    ButtonVariant::Secondary,
                    self.saving,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.change_year(1, cx);
                    }),
                ))
                .child(render_button(
                    "btn-copy-year",
                    &format!("Kopírovat z roku {}", self.year - 1),
                    ButtonVariant::Secondary,
                    has_editing || self.saving,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.request_copy_from_previous_year(cx);
                    }),
                )),
        );

        if self.loading {
            return outer.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Načítání..."),
            );
        }

        if let Some(ref error) = self.error {
            outer = outer.child(
                div()
                    .px_4()
                    .py_3()
                    .bg(rgb(ZfColors::STATUS_RED_BG))
                    .rounded_md()
                    .text_sm()
                    .text_color(rgb(ZfColors::STATUS_RED))
                    .child(error.clone()),
            );
        }

        // Credit cards in a 2-column grid
        outer = outer.child(
            div()
                .flex()
                .gap_4()
                .child(div().flex_1().child(self.render_spouse_card(cx)))
                .child(div().flex_1().child(self.render_children_card(cx))),
        );

        outer = outer.child(
            div()
                .flex()
                .gap_4()
                .child(div().flex_1().child(self.render_personal_card(cx)))
                .child(div().flex_1().child(self.render_deductions_card(cx))),
        );

        // Confirm dialog overlay
        if let Some(ref dialog) = self.confirm_dialog {
            outer = outer.child(dialog.clone());
        }

        outer
    }
}
