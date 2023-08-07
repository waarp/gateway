######################
Gestion des transferts
######################

La commande de gestion des transferts en cours est ``transfer``. Cette commande
doit ensuite être suivie d'une action. La documentation complète de la commande
est disponible :any:`ici <reference-cli-client-transfers>`.

Pour les transferts terminés, la commande est ``history``. La documentation de
la commande est disponible :any:`ici <reference-cli-client-history>`.

.. _user-add-transfer:

Ajouter un transfert
====================

Pour programmer un nouveau transfert, la commande est ``transfer add``. Les
options de commande suivantes doivent être fournies:

- ``-f``: le nom du fichier à transférer
- ``-n``: le nouveau nom du fichier après le transfert (optionnel, par défaut
  le nom reste identique à l'original)
- ``-w``: le sens du transfert (``pull`` ou ``push``)
- ``-p``: le :term:`partenaire` de transfert
- ``-a``: le :term:`compte distant` utilisé pour le transfert
- ``-r``: la :term:`règle` utilisée pour le transfert
- ``-d``: la date du transfert en format `ISO 8601 <https://tools.ietf.org/html/rfc3339>`_
  (optionnel, par défaut le transfert démarre immédiatement)

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' transfer add -f 'exemple.txt' -w 'push' -p 'opensshd' -a 'toto' -r 'règle rebond'

Si les paramètres du transfert sont valides, le transfert sera programmé et
l'identifiant attribué au transfert sera affiché dans la console.


Interrompre/Reprendre un transfert
==================================

Un transfert peut être interrompu avec la commande ``transfer pause``, suivie de
l'identifiant du transfert.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' transfer pause '1234'

Pour reprendre un transfert interrompu, la commande est ``transfer resume``, suivi
de l'identifiant du transfert.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' transfer resume '1234'


Consulter les transferts
========================

Pour lister les transferts en cours, la commande est ``transfer list``. Les
options de commande permettent de filtrer les résultats selon divers critères,
pour plus de détails, voir la :any:`reference
<reference-cli-client-servers-list>`.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' transfers list

Pour les transferts terminés, la commande est ``history list``. La documentation
de la commande est disponible :any:`ici <reference-cli-client-servers-list>`.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' history list

Mettre un transfert en pause
============================

Pour mettre un transfert en pause, la commande est ``transfer pause``, suivie
ensuite de l'identifiant du transfert.

**Exemple**

.. code-block:: shell

   waarp-gateway 'https://admin@127.0.0.1:8080' transfer pause '1234'

Reprendre un transfert arrêté
=============================

Pour reprendre un transfert à l'arrêt, la commande est ``transfer resume``, suivie
ensuite de l'identifiant du transfert. Le transfert reprendra là où il s'était
arrêté.

.. note:: Seuls les transferts en pause ou en erreur peuvent être repris.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' transfer resume '1234'

Annuler un transfert
====================

Pour annuler un transfert, la commande est ``transfer cancel``, suivie ensuite de
l'identifiant du transfert.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' transfer cancel '1234'


Reprogrammer un transfert
=========================

Pour reprogrammer un transfert, la commande est ``history retry``, suivie ensuite
de l'identifiant du transfert. Le transfert recommencera depuis le début.

.. note:: Seuls les transferts terminés peuvent être reprogrammés.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'https://admin@127.0.0.1:8080' transfer retry '1234'
