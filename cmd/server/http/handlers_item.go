package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/naughtygopher/errors"

	"github.com/prashantkr001/template-go/internal/item"
)

func (ht *HTTP) itemRoutes(router chi.Router) {
	router.Post("/items", ht.ErrorHandler(ht.CreateItem))
	router.Get("/items", ht.ErrorHandler(ht.ListItems))
}

func (ht *HTTP) CreateItem(w http.ResponseWriter, req *http.Request) error {
	payload := item.Item{}
	err := json.NewDecoder(req.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "failed to decode request body")
	}

	createdItem, err := ht.apis.ItemCreateIfNotExists(req.Context(), payload)
	if err != nil {
		return err
	}

	jResp, err := json.Marshal(createdItem)
	if err != nil {
		return errors.Wrap(err, "failed to marshal response")
	}

	_, err = w.Write(jResp)
	if err != nil {
		return errors.Wrap(err, "failed to write response")
	}

	return nil
}

func (ht *HTTP) ListItems(w http.ResponseWriter, req *http.Request) error {
	str := req.URL.Query().Get("limit")
	limit, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return errors.InputBodyf("invalid limit provided: %s", str)
	}

	output, err := ht.apis.ItemList(req.Context(), int(limit))
	if err != nil {
		return err
	}

	jResp, err := json.Marshal(output)
	if err != nil {
		return errors.Wrap(err, "failed to marshal response")
	}

	_, err = w.Write(jResp)
	if err != nil {
		return errors.Wrap(err, "failed to write response")
	}
	return nil
}
