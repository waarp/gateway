============================
Afficher un identifiant SMTP
============================

.. program:: waarp-gateway email credential get

Affiche l'identifiant SMTP donné en paramètre.

**Commande**

.. code-block:: shell

   waarp-gateway email credential get "<EMAIL>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway email credential get "waarp@example.com"
