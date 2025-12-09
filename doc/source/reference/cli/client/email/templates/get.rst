============================
Afficher un template d'email
============================

.. program:: waarp-gateway email template get

Affiche le template d'email donné en paramètre.

**Commande**

.. code-block:: shell

   waarp-gateway email template get "<TEMPLATE>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway email template get "alert_erreur"
