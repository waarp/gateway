=======================
Modifier un utilisateur
=======================

.. program:: waarp-gateway user update

.. describe:: waarp-gateway user update <USER>

Remplace les attributs de l'utilisateur demandé avec ceux donnés. Les attributs
omis restent inchangés.

.. option:: -u <USERNAME>, --username=<USERNAME>

   Le nom de l'utilisateur. Doit être unique.

.. option:: -p <PASS>, --password=<PASS>

   Le mot de passe de l'utilisateur.

.. option:: -r <RIGHTS>, --rights=<RIGHTS>

   Les droits de l'utilisateur en format `chmod <https://fr.wikipedia.org/wiki/Chmod#Modes>`_
   (format lettres uniquement). Les cibles possibles pour ces permission sont :

   * **T** pour les transferts
   * **S** pour les serveurs locaux
   * **P** pour les partenaires distants
   * **R** pour les règles de transfert
   * **U** pour les utilisateurs

   Il existe 3 permissions pour chaque cible:

   * **r**: autorisation de lecture
   * **w**: autorisation d'écriture
   * **d**: autorisation de suppression (*Note*: les transferts ne pouvant être
      supprimés, cette autorisation est inconséquente dans leur cas)

   Enfin, il existe 3 opérateurs de changement d'état:

   * **+** pour ajouter un droit aux droits courants
   * **-** pour enlever un droit aux droits courants
   * **=** pour écrases les droits courants

   Ensembles, une cible, un opérateur et les permissions forment un groupe. Les
   groupes doivent être séparé par une virgule `,`.

|

**Exemple**

Pour changer l'utilisateur 'toto', et lui retirer le droit de supprimer des règles
tout en lui ajoutant le droit de modifier les autres utilisateurs; la syntaxe est
la suivante.

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 user update toto -u 'toto2' -p 'sésame2' -r 'R-d,U+rw'