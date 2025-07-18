.. _ref-cmd-waarp-gatewayd-sql:

######################
``waarp-gatewayd sql``
######################

.. program:: waarp-gatewayd sql

``waarp-gatewayd sql`` est une commande permettant d'exécuter directement une
commande SQL sur la base de données de la Gateway.

.. code-block:: shell

   waarp-gatewayd sql "<QUERY>"

Cette commande accepte les options suivantes :

.. option:: --config FILE, -c FILE

   Définit le fichier de configuration à utiliser.

.. option:: --select, -s

   Si présent, indique que la requête est un *SELECT*, et aura donc un retour.
   Le retour de cette commande sera affiché dans la sortie standard. En l'absence
   de ce flag, la requête n'aura pas de retour.

   .. warning::
      Dans le cas d'une requête *SELECT*, penser à bien fixer une limite au nombre
      de résultats. Sinon, la requête risque de prendre longtemps voir même de
      saturer la mémoire du programme.

**Exemple**

.. code-block:: shell

   waarp-gatewayd -c "gatewayd.ini" -s "SELECT * FROM version"