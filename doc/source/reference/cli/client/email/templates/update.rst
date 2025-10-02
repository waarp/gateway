============================
Modifier un template d'email
============================

.. program:: waarp-gateway email template update

Ajoute un nouveau template d'email.

**Commande**

.. code-block:: shell

   waarp-gateway email template update "<TEMPLATE>"

**Options**

.. option:: -n <NAME>, --name=<NAME>

   Le nouveau nom du template. Doit être unique.

.. option:: -s <SUBJECT>, --subject=<SUBJECT>

   Le sujet de l'email. Peut contenir des :ref:`variables de substitution
   <reference-tasks-substitutions>`.

.. option:: -m <TYPE>, --mime-type=<TYPE>

   Le type MIME du corps de l'email. Typiquement soit ``text/plain``, soit
   ``text/html``. Par défaut, ``text/plain`` est utilisé.

.. option:: -b <BODY>, --body=<BODY>

   Le corps de l'email. Peut contenir des :ref:`variables de substitution
   <reference-tasks-substitutions>`. Cette option accepte les chemins de fichiers,
   auquel cas, le contenu du fichier donné sera utilisé comme corps de l'email.

.. option:: -a <ATTACHMENT>, --attachements=<ATTACHMENT>

   Les chemins des fichiers à joindre à l'email en pièces jointes. Répéter
   l'option pour chaque fichier à ajouter.

**Exemple**

.. code-block:: shell

   waarp-gateway email template update "ancien_template" -n "alerte_erreur" -s "Notification d'erreur de transfert" -m "text/plain" -b "Le transfert n°#TRANSFERID# a échoué le #DATE# à #HOUR# avec l'erreur #ERRORMSG#" -a "gateway.log"
