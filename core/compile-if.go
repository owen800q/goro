package core

import (
	"io"

	"github.com/MagicalTux/goro/core/tokenizer"
)

type runnableIf struct {
	cond Runnable
	yes  Runnable
	no   Runnable
	l    *Loc
}

func (r *runnableIf) Run(ctx Context) (l *ZVal, err error) {
	t, err := r.cond.Run(ctx)
	if err != nil {
		return nil, err
	}
	t, err = t.As(ctx, ZtBool)
	if err != nil {
		return nil, err
	}

	if t.Value().(ZBool) {
		return r.yes.Run(ctx)
	} else if r.no != nil {
		return r.no.Run(ctx)
	} else {
		return nil, nil
	}
}

func (r *runnableIf) Loc() *Loc {
	return r.l
}

func (r *runnableIf) Dump(w io.Writer) error {
	_, err := w.Write([]byte("if ("))
	if err != nil {
		return err
	}
	err = r.cond.Dump(w)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(") {"))
	if err != nil {
		return err
	}
	err = r.yes.Dump(w)
	if err != nil {
		return err
	}
	if r.no != nil {
		_, err = w.Write([]byte("} else {"))
		if err != nil {
			return err
		}
		err = r.no.Dump(w)
		if err != nil {
			return err
		}
	}
	_, err = w.Write([]byte{';'})
	return err
}

func compileIf(i *tokenizer.Item, c compileCtx) (Runnable, error) {
	// T_IF (expression) ...? else ...?

	// parse if expression
	i, err := c.NextItem()
	if err != nil {
		return nil, err
	}
	if !i.IsSingle('(') {
		return nil, i.Unexpected()
	}

	r := &runnableIf{l: MakeLoc(i.Loc())}
	r.cond, err = compileExpr(nil, c)
	if err != nil {
		return nil, err
	}

	// check for )
	i, err = c.NextItem()
	if err != nil {
		return nil, err
	}
	if !i.IsSingle(')') {
		return nil, i.Unexpected()
	}

	// check for next if ':'
	i, err = c.NextItem()
	if err != nil {
		return nil, err
	}
	if i.IsSingle(':') {
		// parse expression until endif
		// See: http://php.net/manual/en/control-structures.alternative-syntax.php
		r.yes, err = compileBase(nil, c)
		if err != nil {
			return nil, err
		}

		i, err = c.NextItem()
		if err != nil {
			return r, err
		}

		switch i.Type {
		case tokenizer.T_ELSEIF:
			r.no, err = compileIf(nil, c)
			if err != nil {
				return nil, err
			}
		case tokenizer.T_ELSE:
			i, err = c.NextItem()
			if err != nil {
				return r, err
			}
			if !i.IsSingle(':') {
				return nil, i.Unexpected()
			}
			r.no, err = compileBase(nil, c)

			// then we should be getting a endif
			i, err = c.NextItem()
			if err != nil {
				return r, err
			}
			if i.Type != tokenizer.T_ENDIF {
				return nil, i.Unexpected()
			}
			fallthrough
		case tokenizer.T_ENDIF:
			// end of if
			i, err = c.NextItem()
			if err != nil {
				return r, err
			}
			if !i.IsSingle(';') {
				return nil, i.Unexpected()
			}
		default:
			return nil, i.Unexpected()
		}
	} else {
		c.backup()

		// parse expression normally
		r.yes, err = compileBaseSingle(nil, c)
		if err != nil {
			return nil, err
		}

		i, err = c.NextItem()
		if err != nil {
			return r, err
		}

		// check for else or elseif
		switch i.Type {
		case tokenizer.T_ELSEIF:
			r.no, err = compileIf(nil, c)
			if err != nil {
				return nil, err
			}
		case tokenizer.T_ELSE:
			// parse else
			r.no, err = compileBaseSingle(nil, c)
			if err != nil {
				return nil, err
			}
		default:
			c.backup()
		}
	}

	return r, nil
}
