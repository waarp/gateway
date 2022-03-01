############################
Configuration des containers
############################


Lancement d'une image
=====================



Variables d'environnement
=========================


Synchronisation depuis manager
------------------------------


Séquence de démarrage
=====================

Lors du lancement du container, plusieurs vérifications et opérations de
configurations sont réalisées avant le lancement de Waarp Gateway.

Voici le processus suivi lors du démarrage :


.. uml::

   start

   :Lecture du fichier de configuration;

   :Mise à jour de la configuration
   à partir de l'environnement;

   :Écriture du fichier de configuration;

   if (Utilisation de Manager) then (oui)
      :Vérification des variables
      d'environnement requises;

      if (Certificats présents) then (oui)
      else (non)
         :Génération de certificats
         auto-signés;
      endif

      if (Gateway est déclarée dans Manager) then (oui)
      else (non)
         :Vérification des variables
         d'environnement requises
         pour l'enregistrement;

         :Enregistrement dans Manager;
      endif

      :Téléchargement de la configuration
      depuis Manager;

      :Import de la configuration
      dans Gatewayd;
   else (non)
   endif

   :Lancement de Gatewayd;

   stop


