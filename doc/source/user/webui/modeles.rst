.. _user-webui-email-templates:

################
Modèles d'e-mail
################

Section *Traitements*, adresse ``/webui/email-templates``. La consultation
requiert le droit de lecture sur les règles ; la création et la modification,
le droit d'écriture ; la suppression, le droit de suppression.

Un modèle d'e-mail est le message qu'envoie le traitement
:doc:`/reference/tasks/email` : un nom (unique), un objet, un format (texte ou
HTML), un corps et, au besoin, des pièces jointes désignées par leur chemin.
Un traitement ``EMAIL`` désigne le modèle par son nom (paramètre ``template``) ;
il y ajoute l'identifiant SMTP de l'expéditeur et les destinataires.

L'objet, le corps et les chemins des pièces jointes acceptent les
:ref:`marqueurs de substitution <reference-tasks-substitutions>` des
traitements (``#TRUEFILENAME#``, ``#TRANSFERID#``, ``#TIMESTAMP(DD/MM/YYYY)#``,
``#TI_<clé>#``…), remplacés par les valeurs du transfert au moment de l'envoi.


La liste
========

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Colonne
     - Contenu
   * - Modèle
     - Le nom du modèle ; il ouvre la fiche du modèle.
   * - Objet
     - L'objet du message, ses marqueurs tels quels.
   * - Format
     - *Texte* ou *HTML* ; un type MIME posé par l'API REST s'affiche tel quel.
   * - Pièces jointes
     - Le nombre de pièces jointes, ou *Aucune*.
   * - Utilisé par
     - Le nombre de règles dont les traitements ``EMAIL`` désignent le modèle.
   * - Actions
     - Le menu « ⋯ » de la ligne : voir la fiche, modifier le modèle (droit
       d'écriture), le supprimer (droit de suppression).

Les colonnes *Modèle* et *Objet* se trient d'un clic sur leur en-tête (par
défaut, le nom croissant). La recherche filtre les modèles par leur nom
(:ref:`règle commune <user-webui-search>`) et propose les noms existants. Le pied de liste permet de
changer de page et de choisir le nombre de lignes par page. La recherche, le tri et
la page sont portés par l'adresse de la page ; le tri et le nombre de lignes par
page sont conservés pour la durée de la session du navigateur, comme pour les
autres listes.

Sans droit d'écriture, le bouton *Ajouter un modèle* et les actions qui
modifient un modèle n'apparaissent pas ; le serveur refuse de toute façon une
action sans le droit correspondant.


Fiche d'un modèle
=================

La fiche d'un modèle, adresse ``/webui/email-templates/<id>``, montre en
lecture :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Contenu
   * - Message
     - L'objet, ses marqueurs tels quels, et le format.
   * - Aperçu
     - Le message tel que le destinataire le reçoit (voir ci-dessous).
   * - Pièces jointes
     - Les chemins des pièces jointes, un par ligne, ou *Aucune*.
   * - Utilisé par
     - Les règles dont les traitements ``EMAIL`` désignent le modèle, chacune
       en carte avec son sens (envoi ou réception) ; le nom de la règle ouvre ses
       traitements, où le modèle se change.


L'aperçu
--------

L'aperçu montre le message tel que le traitement ``EMAIL`` l'envoie : l'objet,
les pièces jointes et le corps, les marqueurs remplacés par le code même du
traitement. Un marqueur que les traitements ne connaissent pas reste tel quel,
comme à l'envoi. L'expéditeur et les destinataires appartiennent au
traitement, pas au modèle : l'aperçu les nomme sans les afficher.

*Valeurs des variables* choisit d'où viennent les valeurs :

- *Valeurs d'exemple* : un transfert fictif en erreur, avec les dossiers de la
  passerelle ;
- l'un des cinq derniers transferts, en cours ou terminés, des règles qui
  emploient le modèle : ses vraies valeurs, ce qui montre le message qu'il a
  produit ou aurait produit.

Tant qu'aucun transfert n'a employé le modèle, l'aperçu prend les valeurs
d'exemple et le dit. *Repérer les variables* montre chaque valeur remplacée
en étiquette.

Le corps s'affiche dans un cadre isolé : un modèle HTML n'y exécute aucun
script.

Si le modèle contient un marqueur que les traitements ne prennent pas en
charge, l'aperçu donne l'erreur avec laquelle le traitement échouerait.


Créer ou modifier un modèle
===========================

Le bouton *Ajouter un modèle* ouvre le formulaire de création ; *Modifier le
modèle* ouvre le même formulaire. Il se compose de quatre blocs :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Champs
   * - Modèle
     - Le nom, unique, et le format : *HTML* (proposé à la création) ou *Texte*.
       Un modèle enregistré par l'API REST avec un autre type MIME le garde.
   * - Message
     - L'objet et le corps. Un corps HTML s'écrit dans un éditeur (voir
       ci-dessous) ; un corps texte, dans une zone de texte. Changer de format
       convertit le corps : les paragraphes d'un texte deviennent des
       paragraphes HTML, un HTML devient son texte. Le corps peut aussi se charger
       depuis un fichier HTML (.html, .htm) ou texte (.txt), en UTF-8, de 1 Mo
       au plus : son contenu remplace le corps et le format suit son extension ;
       il se relit avant validation. Un fichier d'un autre type, trop gros ou
       d'un autre encodage est refusé, avec sa raison.
   * - Pièces jointes
     - Les chemins des fichiers à joindre, un par ligne. Le fichier doit
       exister au moment de l'envoi, sinon le traitement échoue.
   * - Aperçu
     - Le message que le formulaire tient, mis à jour pendant la saisie.

Quand des règles emploient le modèle, son nom ne se modifie pas : leurs
traitements le désignent. Le formulaire le montre en lecture.


L'éditeur HTML
--------------

L'éditeur affiche le corps mis en forme. Sa barre d'outils propose le gras,
l'italique, le souligné, le titre, le paragraphe, les listes à puces et
numérotées, le lien, l'effacement de la mise en forme et *Source HTML*, qui
montre et modifie le code HTML. Au clavier, la barre d'outils est un seul arrêt
de tabulation, les flèches passent d'un outil à l'autre.

Un texte collé dans l'éditeur garde sa structure (paragraphes, titres, listes,
liens, gras et italique) et perd le reste (styles, scripts, attributs) ; un
code HTML complet se colle dans *Source HTML*. Le corps s'affiche dans un cadre
isolé : rien de ce qu'il contient ne s'exécute.


Insérer une variable
--------------------

Dans l'objet, le corps et les pièces jointes, taper ``#`` ouvre la liste des
variables à l'endroit du curseur. La frappe suivante la filtre, par le nom du
marqueur puis par son libellé, sans tenir compte des majuscules ni des accents
(``#fil`` propose ``#FILESIZE#``, ``#FILEEXTENSION#``, ``#TRUEFILENAME#``…). Les
flèches parcourent la liste, *Entrée* ou *Tab* insère le marqueur, *Échap*
ferme la liste et garde le ``#`` tapé. Le bouton *Variables* ouvre la même
liste.

La liste groupe les marqueurs que les traitements prennent en charge, chacun
avec son libellé. Deux sortes de marqueurs demandent un paramètre, inséré
sélectionné pour être remplacé :

- les horodatages (``#TIMESTAMP(…)#``, ``#STARTTIMESTAMP(…)#``) : tant que le
  curseur est dans le format, la liste propose ses éléments (``YYYY``, ``MM``,
  ``DD``, ``HH``, ``mm``…) avec la valeur qu'ils donnent maintenant ;
- les informations de transfert (``#TI_<clé>#``) : la clé est à saisir.

Dans l'éditeur, un marqueur s'affiche comme une étiquette.

*Valider* enregistre le modèle avec une notification et ramène à la liste
(création) ou à la fiche du modèle (modification). Un formulaire refusé est
réaffiché avec, sous chaque champ fautif, ce qu'il faut corriger. Le formulaire
refuse un marqueur que les traitements ne connaissent pas (``#FOO#``) ou ne
prennent pas en charge (``#ARCHPATH#``…) : le premier resterait tel quel dans
le message envoyé, le second ferait échouer le traitement.


Supprimer un modèle
===================

La suppression d'un modèle qu'aucune règle n'emploie se confirme sur une page
qui rappelle son nom et son objet.

Un modèle que des règles emploient ne se supprime pas : leurs traitements
``EMAIL`` échoueraient au prochain transfert. La page le dit et liste ces
règles, dont le nom ouvre les traitements : il faut d'abord y changer le
modèle.


Et par l'API
============

Les mêmes modèles se gèrent par l':ref:`API REST <reference-rest-email>`
(``/api/email/templates``) et par la :ref:`ligne de commande
<ref-cli-client-email>` (``waarp-gateway email template``). Un modèle créé par
l'un s'affiche et se modifie par les autres ; l'API REST accepte un marqueur
que l'écran refuse, et l'aperçu montre alors ce qu'il produit.

L'ancienne page *Templates d'email*, qui renvoyait à la ligne de commande, mène
désormais à cet écran.
