======================
Ajouter un utilisateur
======================

.. program:: waarp-gateway user add

Ajoute un nouvel utilisateur avec les identifiants donnés.

**Commande**

.. code-block:: shell

   waarp-gateway user add

**Options**

.. option:: -u <USERNAME>, --username=<USERNAME>

   Le nom du nouvel utilisateur créé. Doit être unique.

.. option:: -p <PASS>, --password=<PASS>

   Le mot de passe du nouvel utilisateur.

.. option:: -r <RIGHTS>, --rights=<RIGHTS>

   Les droits de l'utilisateur en format `chmod
   <https://fr.wikipedia.org/wiki/Chmod#Modes>`_ (format lettres uniquement).
   Les cibles possibles pour ces permission sont :

   * ``T`` pour les transferts
   * ``S`` pour les serveurs locaux
   * ``P`` pour les partenaires distants
   * ``R`` pour les règles de transfert
   * ``U`` pour les utilisateurs
   * ``A`` pour l'administration de Waarp Gateway

   Il existe 3 permissions pour chaque cible:

   * ``r``: autorisation de lecture
   * ``w``: autorisation d'écriture
   * ``d``: autorisation de suppression (*Note*: les transferts ne pouvant être
     supprimés, cette autorisation est inconséquente dans leur cas)

   Enfin, il existe 3 opérateurs de changement d'état:

   * ``+`` pour ajouter un droit aux droits courants
   * ``-`` pour enlever un droit aux droits courants
   * ``=`` pour écrases les droits courants

   *Note*: Étant donné que l'utilisateur n'a pas de droits courants,les opérateurs
   *+* et *-* n'ont pas vraiment de sens avec la commande de création d'utilisateur.

   Ensembles, une cible, un opérateur et les permissions forment un groupe. Les
   groupes doivent être séparé par une virgule ``,``.

**Exemple**

Pour créer un utilisateur ayant le droit d'ajouter des transferts, consulter les
serveurs/partenaires, et d'ajouter/supprimer des règles; la syntaxe est la suivante.

.. code-block:: shell

   waarp-gateway user add -u 'toto' -p 'sésame' -r 'T=rw,S=r,P=r,R=rwd'
