============================
Annuler plusieurs transferts
============================

.. program:: waarp-gateway transfer cancel-all

Annule plusieurs transferts d'un coup. Avec la seule option ``--target``,
annule tous les transferts (non-terminés) ayant le statut renseigné, comme
auparavant. Avec les options de filtre ci-dessous, annule les transferts qui
les remplissent, en une seule requête : les transferts que ``transfer list``
afficherait avec les mêmes filtres, moins ceux exclus par ``--except`` ;
``--target``, si elle est renseignée, restreint l'ensemble à ses statuts. Les
transferts annulés sont déplacés dans l'historique de transfert.

Avec des options de filtre, la commande affiche combien de transferts ont été
annulés, combien ont été laissés de côté et pourquoi (un transfert terminé ne
s'annule plus, un transfert qui s'exécute sur un autre nœud ou qui ne s'arrête
pas à temps).

**Commande**

.. code-block:: shell

   waarp-gateway transfer cancel-all

**Options**

.. option:: -t <TARGET>, --target <TARGET>

   Filtre les transferts à annuler par statut. Cette option est requise en
   l'absence de toute autre option. Les valeurs possibles pour cette option
   sont :

   - annuler les transferts non-démarrés (``planned``)
   - annuler les transferts en cours (``running``)
   - annuler les transferts en pause (``paused``)
   - annuler les transferts interrompus (``interrupted``)
   - annuler les transferts en erreur (``error``)
   - annuler tous les transferts non-terminés (``all``)

.. option:: -r <RULE>, --rule=<RULE>

   Restreint l'ensemble aux transferts utilisant la règle renseignée. L'option
   peut être renseignée plusieurs fois pour plusieurs règles.

.. option:: --status=<STATUS>

   Restreint l'ensemble aux transferts ayant le statut renseigné. L'option peut
   être renseignée plusieurs fois pour plusieurs statuts. Les statuts valides
   sont : ``PLANNED``, ``RUNNING``, ``PAUSED``, ``INTERRUPTED`` et ``ERROR``.

.. option:: -q <ACCOUNT>, --requester=<ACCOUNT>

   Restreint l'ensemble aux transferts dont le compte demandeur porte le nom
   renseigné. L'option peut être renseignée plusieurs fois pour plusieurs
   comptes.

.. option:: --requested=<AGENT>

   Restreint l'ensemble aux transferts demandés au serveur ou partenaire
   renseigné. L'option peut être renseignée plusieurs fois pour plusieurs
   serveurs ou partenaires.

.. option:: --start=<DATE>

   Restreint l'ensemble aux transferts dont la date de début est postérieure ou
   égale à celle renseignée (:rfc:`3339`).

.. option:: -f <FOLLOW_ID>, --follow-id=<FOLLOW_ID>

   Restreint l'ensemble aux transferts ayant l'ID de flux renseigné.

.. option:: --except=<ID>

   Exclut de l'ensemble le transfert portant cet identifiant. L'option peut
   être renseignée plusieurs fois.

**Exemples**

.. code-block:: shell

   waarp-gateway transfer cancel-all -t error

.. code-block:: shell

   waarp-gateway transfer cancel-all -t error --rule commandes --except 12
