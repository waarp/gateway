============================
Annuler plusieurs transferts
============================

.. program:: waarp-gateway transfer cancel-all

Annule tous les transferts (non-terminés) ayant le statut renseigné en option.
Ces transferts seront annulés, puis déplacés dans l'historique de transfert.

**Commande**

.. code-block:: shell

   waarp-gateway transfer cancel-all

**Options**

.. option:: -t <TARGET>, --target <TARGET>

   Filtre les transferts à annuler par statut. Cette option est requise. Les
   valeurs possible pour cette option sont :

   - annuler les transferts non-démarrés (``planned``)
   - annuler les transferts en cours (``running``)
   - annuler les transferts en pause (``paused``)
   - annuler les transferts interrompus (``interrupted``)
   - annuler les transferts en erreur (``error``)
   - annuler tous les transferts non-terminés (``all``)

**Exemple**

.. code-block:: shell

   waarp-gateway transfer cancel-all -t error
